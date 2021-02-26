# bench-post

## copy bench.go to lotus/cmd/lotus-storage-miner
```
cp bench.go lotus/cmd/lotus-storage-miner
```

## add benchCmd to lotus/cmd/lotus-storage-miner/main.go
```
func main() {
	build.RunningNodeType = build.NodeMiner

	lotuslog.SetupLogLevels()

	local := []*cli.Command{
		initCmd,
		runCmd,
		stopCmd,
		configCmd,
		benchCmd,   //add there
		loadingCmd,
		backupCmd,
    ....
```

# Use
### --file sector id file
```
3000
3001
```

### --db  lotus-miner home
```
LOTUS_MINER_PATH
```

### bench post
```
./lotus-miner bench --file id.txt --db ~/.lotus-miner
```
