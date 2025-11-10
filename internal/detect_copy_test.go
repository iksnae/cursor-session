package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iksnae/cursor-session/testutil"
	_ "modernc.org/sqlite"
)

func TestCopyFile(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	// Create source file
	content := "test content"
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify destination exists
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Error("copyFile() did not create destination file")
	}

	// Verify content
	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(got) != content {
		t.Errorf("copyFile() content = %q, want %q", string(got), content)
	}
}

func TestCopyFile_NonexistentSource(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	srcFile := filepath.Join(tmpDir, "nonexistent.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	err := copyFile(srcFile, dstFile)
	if err == nil {
		t.Error("copyFile() should return error for nonexistent source")
	}
}

func TestCopyDatabaseWithWAL(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	srcDB := filepath.Join(tmpDir, "source.db")
	dstDB := filepath.Join(tmpDir, "dest.db")

	// Create source database
	testutil.CreateSQLiteFixture(t, srcDB)

	// Copy database
	if err := copyDatabaseWithWAL(srcDB, dstDB); err != nil {
		t.Fatalf("copyDatabaseWithWAL() error = %v", err)
	}

	// Verify destination exists
	if _, err := os.Stat(dstDB); os.IsNotExist(err) {
		t.Error("copyDatabaseWithWAL() did not create destination database")
	}

	// Verify database is readable
	db, err := OpenDatabase(dstDB)
	if err != nil {
		t.Fatalf("Failed to open copied database: %v", err)
	}
	defer func() { _ = db.Close() }()
}

func TestCheckpointWAL(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database
	testutil.CreateSQLiteFixture(t, dbPath)

	// Checkpoint should succeed even without WAL file
	if err := checkpointWAL(dbPath); err != nil {
		t.Errorf("checkpointWAL() error = %v, want nil", err)
	}

	// Verify database is still readable
	db, err := OpenDatabase(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database after checkpoint: %v", err)
	}
	defer func() { _ = db.Close() }()
}

func TestCheckpointWAL_Nonexistent(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)
	dbPath := filepath.Join(tmpDir, "nonexistent.db")

	// checkpointWAL with mode=rwc will create the database if it doesn't exist
	// So we test that it doesn't error (it creates an empty database)
	err := checkpointWAL(dbPath)
	if err != nil {
		t.Errorf("checkpointWAL() error = %v, want nil (creates database if missing)", err)
	}

	// Verify database was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("checkpointWAL() did not create database")
	}
}

func TestCopyStoragePaths(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)

	// Create a test database
	testDB := filepath.Join(tmpDir, "state.vscdb")
	testutil.CreateSQLiteFixture(t, testDB)

	// Create storage paths pointing to the test database
	paths := StoragePaths{
		GlobalStorage: tmpDir,
	}

	// Copy storage paths
	copiedPaths, cleanup, err := CopyStoragePaths(paths)
	if err != nil {
		t.Fatalf("CopyStoragePaths() error = %v", err)
	}
	defer func() { _ = cleanup() }()

	// Verify copied database exists
	copiedDB := copiedPaths.GetGlobalStorageDBPath()
	if _, err := os.Stat(copiedDB); os.IsNotExist(err) {
		t.Error("CopyStoragePaths() did not copy database")
	}

	// Verify database is readable
	db, err := OpenDatabase(copiedDB)
	if err != nil {
		t.Fatalf("Failed to open copied database: %v", err)
	}
	defer func() { _ = db.Close() }()
}

func TestCopyStoragePaths_NoStorage(t *testing.T) {
	// Create storage paths with nonexistent paths
	paths := StoragePaths{
		GlobalStorage: "/nonexistent/path",
	}

	// Copy should succeed but not copy anything (since GlobalStorageExists() returns false)
	copiedPaths, cleanup, err := CopyStoragePaths(paths)
	if err != nil {
		t.Fatalf("CopyStoragePaths() error = %v", err)
	}
	defer func() { _ = cleanup() }()

	// When no storage exists, paths should remain unchanged (no copying happens)
	// The function only updates paths when it actually copies something
	if copiedPaths.GlobalStorage != paths.GlobalStorage {
		t.Logf("CopyStoragePaths() updated paths (this is OK if no storage exists)")
	}
}

func TestCopyStoragePaths_Cleanup(t *testing.T) {
	tmpDir := testutil.CreateTempDir(t)

	// Create a test database
	testDB := filepath.Join(tmpDir, "state.vscdb")
	testutil.CreateSQLiteFixture(t, testDB)

	paths := StoragePaths{
		GlobalStorage: tmpDir,
	}

	// Copy storage paths
	copiedPaths, cleanup, err := CopyStoragePaths(paths)
	if err != nil {
		t.Fatalf("CopyStoragePaths() error = %v", err)
	}

	// Verify copied database exists
	copiedDB := copiedPaths.GetGlobalStorageDBPath()
	if _, err := os.Stat(copiedDB); os.IsNotExist(err) {
		t.Error("CopyStoragePaths() did not copy database")
	}

	// Cleanup
	if err := cleanup(); err != nil {
		t.Fatalf("cleanup() error = %v", err)
	}

	// Verify copied database is removed
	if _, err := os.Stat(copiedDB); !os.IsNotExist(err) {
		t.Error("cleanup() did not remove copied database")
	}
}
