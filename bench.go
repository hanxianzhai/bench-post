package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"github.com/filecoin-project/go-bitfield"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	lcli "github.com/filecoin-project/lotus/cli"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper"
	"github.com/filecoin-project/lotus/extern/sector-storage/ffiwrapper/basicfs"
	proof2 "github.com/filecoin-project/specs-actors/v2/actors/runtime/proof"
	"golang.org/x/xerrors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"

	addr "github.com/filecoin-project/go-address"
)

func ReadSids(fileName string) ([]int, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer  f.Close()
	buf := bufio.NewReader(f)

	var sids []int
	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		line = strings.TrimSpace(line)
		sid, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}
		sids = append(sids, sid)
	}
	return sids, nil
}

func allSectors(ctx context.Context, api api.FullNode, maddr addr.Address, allSectors bitfield.BitField) ([]proof2.SectorInfo, error) {
	sset, err := api.StateMinerSectors(ctx, maddr, &allSectors, types.EmptyTSK)
	if err != nil {
		return nil, err
	}

	if len(sset) == 0 {
		return nil, nil
	}

	substitute := proof2.SectorInfo{
		SectorNumber: sset[0].SectorNumber,
		SealedCID:    sset[0].SealedCID,
		SealProof:    sset[0].SealProof,
	}

	sectorByID := make(map[uint64]proof2.SectorInfo, len(sset))
	for _, sector := range sset {
		sectorByID[uint64(sector.SectorNumber)] = proof2.SectorInfo{
			SectorNumber: sector.SectorNumber,
			SealedCID:    sector.SealedCID,
			SealProof:    sector.SealProof,
		}
	}

	proofSectors := make([]proof2.SectorInfo, 0, len(sset))
	if err := allSectors.ForEach(func(sectorNo uint64) error {
		if info, found := sectorByID[sectorNo]; found {
			proofSectors = append(proofSectors, info)
		} else {
			proofSectors = append(proofSectors, substitute)
		}
		return nil
	}); err != nil {
		return nil, xerrors.Errorf("iterating partition sector bitmap: %w", err)
	}

	return proofSectors, nil
}


func benchPost(cctx *cli.Context, sids[]int, dir string) error {
	fmt.Printf("bench %v sectors\n", len(sids))
	sectors := bitfield.New()
	for _, id := range sids {
		sectors.Set(uint64(id))
	}

	nodeApi, closer, err := lcli.GetStorageMinerAPI(cctx)
	if err != nil {
		return err
	}
	defer closer()

	api, acloser, err := lcli.GetFullNodeAPI(cctx)
	if err != nil {
		return err
	}
	defer acloser()

	ctx := lcli.ReqContext(cctx)

	maddr, err := getActorAddress(ctx, nodeApi, cctx.String("actor"))
	if err != nil {
		return err
	}

	amid, err := addr.IDFromAddress(maddr)
	if err != nil {
		return err
	}
	mid := abi.ActorID(amid)

	sis, err := allSectors(ctx, api, maddr, sectors)
	if err != nil {
		return err
	}

	sbfs := &basicfs.Provider{
		Root: dir,
	}

	sb, err := ffiwrapper.New(sbfs)
	if err != nil {
		return err
	}

	var challenge [32]byte
	rand.Read(challenge[:])

	_, _, err1 := sb.GenerateWindowPoSt(ctx, mid, sis, challenge[:])
	if err1 != nil {
		return err1
	}

	return nil
}

var benchCmd = &cli.Command{
	Name:  "bench",
	Usage: "bench post",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "file",
			Usage: "bench sector ids",
			Value: "",
		},
		&cli.StringFlag{
			Name: "db",
			Usage: "bench home",
			Value: "",
		},
	},
	Action: func(cctx *cli.Context) error {
		fi := cctx.String("file")
		if fi == "" {
			return fmt.Errorf("input file")
		}

		if cctx.String("db")  == ""{
			return fmt.Errorf("input db")
		}

		sids, err := ReadSids(fi)
		if err != nil {
			return err
		}

		return benchPost(cctx, sids, cctx.String("db"))
	},
}
