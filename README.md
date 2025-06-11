# gobackup-app

### 1. Project Structure
```
gobackup-app/
├── cmd/
│   └── main.go                     # CLI entry point
├── internal/
│   ├── watcher/
│   │   └── watcher.go              # File system monitoring
│   ├── backup/
│   │   ├── chunker.go              # File chunking logic
│   │   ├── compressor.go           # Compression handling
│   │   └── engine.go               # Backup orchestration
│   │   └── chunker.go              # Chunk model
│   ├── restore/
│   │   └── engine.go               # Restore logic
│   ├── metadata/
│   │   └── manager.go              # Metadata and state tracking
│   │   └── file_actions.go         # Helper for file operations
│   └── utils/
│       ├── hash.go                 # File hashing utilities
│       └── filesystem.go           # File system helpers
├── pkg/
│   └── models/
│       └── types.go                # Data structures
└── go.mod
```


```
To Run: do the following: 

❯ go build -o gobackup-app cmd/main.go                                                                                                                         
❯ ./gobackup-app --watch /Users/soujanyanamburi/Projects/gobackup-app/test --backup /Users/soujanyanamburi/Projects/gobackup-app/test-backup --refresh 10
❯ ./gobackup-app --restore --backup /Users/soujanyanamburi/Projects/gobackup-app/test-backup --target /Users/soujanyanamburi/Projects/gobackup-app/test-restore

Default chunk size: 5MB, can be changed in chunk_model.go as per your convenience


Extra things: 
Run help to see what's in store :)) 


```