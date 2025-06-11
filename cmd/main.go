package main

import (
	"context"
	"fmt"
	"gobackup/internal/backup"
	"gobackup/internal/restore"
	"gobackup/internal/watcher"
	"gobackup/pkg/models"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	watchPath   string
	backupPath  string
	targetPath  string
	refreshRate int
	restoreMode bool
	listMode    bool
	verifyMode  bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "gobackup-app",
		Short: "A file backup and restore system",
		Long:  "A comprehensive file backup system with real-time monitoring and chunked storage",
		Run:   runApp,
	}

	rootCmd.Flags().StringVar(&watchPath, "watch", "", "Directory to watch for changes")
	rootCmd.Flags().StringVar(&backupPath, "backup", "", "Directory to store backup files")
	rootCmd.Flags().StringVar(&targetPath, "target", "", "Target directory for restore (restore mode only)")
	rootCmd.Flags().IntVar(&refreshRate, "refresh", 300, "Full scan interval in seconds")
	rootCmd.Flags().BoolVar(&restoreMode, "restore", false, "Enable restore mode")
	rootCmd.Flags().BoolVar(&listMode, "list", false, "List files in backup")
	rootCmd.Flags().BoolVar(&verifyMode, "verify", false, "Verify backup integrity")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
func runApp(cmd *cobra.Command, args []string) {
	if backupPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --backup path is required\n")
		printUsageExamples()
		os.Exit(1)
	}

	modeCount := 0
	if listMode {
		modeCount++
	}
	if verifyMode {
		modeCount++
	}
	if restoreMode {
		modeCount++
	}
	if watchPath != "" {
		modeCount++
	}

	if modeCount == 0 {
		fmt.Fprintf(os.Stderr, "Error: You must specify one operation mode\n")
		printUsageExamples()
		os.Exit(1)
	}

	if modeCount > 1 {
		fmt.Fprintf(os.Stderr, "Error: Only one operation mode can be specified at a time\n")
		printUsageExamples()
		os.Exit(1)
	}

	if listMode {
		if err := listBackupFiles(); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing files: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if verifyMode {
		if err := verifyBackup(); err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying backup: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if restoreMode {
		if targetPath == "" {
			fmt.Fprintf(os.Stderr, "Error: --target path is required for restore mode\n")
			printUsageExamples()
			os.Exit(1)
		}
		if err := runRestore(); err != nil {
			fmt.Fprintf(os.Stderr, "Error during restore: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if watchPath != "" {
		if err := runBackup(); err != nil {
			fmt.Fprintf(os.Stderr, "Error during backup: %v\n", err)
			os.Exit(1)
		}
		return
	}
}
func printUsageExamples() {
	fmt.Fprintf(os.Stderr, `
Usage Examples:
===============

1. Start backup monitoring (watch mode):
   %s --watch /path/to/watch --backup /path/to/backup --refresh 60

2. Restore from backup:
   %s --restore --backup /path/to/backup --target /path/to/restore

3. List files in backup:
   %s --list --backup /path/to/backup

4. Verify backup integrity:
   %s --verify --backup /path/to/backup

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}
func runBackup() error {
	log.Printf("Starting backup system...")
	log.Printf("Watch path: %s", watchPath)
	log.Printf("Backup path: %s", backupPath)
	log.Printf("Refresh rate: %d seconds", refreshRate)

	if _, err := os.Stat(watchPath); os.IsNotExist(err) {
		return fmt.Errorf("watch path does not exist: %s", watchPath)
	}

	engine := backup.NewEngine(watchPath, backupPath)
	if err := engine.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize backup engine: %w", err)
	}

	w, err := watcher.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer w.Close()

	if err := w.AddWatch(watchPath); err != nil {
		return fmt.Errorf("failed to add watch path: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := engine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backup engine: %w", err)
	}

	w.Start()

	log.Println("Performing initial full backup...")
	if err := engine.PerformFullBackup(); err != nil {
		log.Printf("Warning: initial backup failed: %v", err)
	}

	refreshTicker := time.NewTicker(time.Duration(refreshRate) * time.Second)
	defer refreshTicker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("Backup system started. Press Ctrl+C to stop.")

	for {
		select {
		case <-sigChan:
			log.Println("Shutdown signal received...")
			engine.Shutdown()
			return nil

		case event := <-w.Changes():
			changes := []models.FileChange{
				{
					Path:      event.Path,
					Operation: event.Operation,
				},
			}

			if err := engine.ProcessChanges(changes); err != nil {
				log.Printf("Error processing changes: %v", err)
			}

		case err := <-w.Errors():
			log.Printf("Watcher error: %v", err)

		case <-refreshTicker.C:
			log.Println("Performing periodic full backup...")
			if err := engine.PerformFullBackup(); err != nil {
				log.Printf("Periodic backup failed: %v", err)
			}
		}
	}
}

func runRestore() error {
	log.Printf("Starting restore operation...")
	log.Printf("Backup path: %s", backupPath)
	log.Printf("Target path: %s", targetPath)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup path does not exist: %s", backupPath)
	}

	engine, err := restore.NewEngine(backupPath, targetPath)
	if err := engine.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize restore engine: %w", err)
	}
	engine.ListFiles()
	if err != nil {
		return fmt.Errorf("failed to initialize restore engine: %w", err)
	}

	if err := engine.RestoreAll(); err != nil {
		return fmt.Errorf("restore operation failed: %w", err)
	}

	log.Println("Restore operation completed successfully!")
	return nil
}

func listBackupFiles() error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup path does not exist: %s", backupPath)
	}

	engine, err := restore.NewEngine(backupPath, "")
	if err := engine.InitializeWithoutTarget(); err != nil {
		return fmt.Errorf("failed to initialize restore engine: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to initialize engine: %w", err)
	}

	return engine.ListFiles()
}

func verifyBackup() error {
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup path does not exist: %s", backupPath)
	}

	engine, err := restore.NewEngine(backupPath, "")
	if err := engine.InitializeWithoutTarget(); err != nil {
		return fmt.Errorf("failed to initialize restore engine: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to initialize engine: %w", err)
	}

	if err := engine.ValidateBackup(); err != nil {
		return fmt.Errorf("backup validation failed: %w", err)
	}

	fmt.Println("Backup verification completed successfully!")
	return nil
}
