package node

import (
	"io/ioutil"
	"os"
	"testing"

	cfg "github.com/bytom/config"
	"fmt"
)

func TestNodeUsedDataDir(t *testing.T) {
	dir, err := ioutil.TempDir("", "temp")
	if err != nil {
		t.Fatalf("failed to create temporary data directory: %v", err)
	}
	defer os.RemoveAll(dir)
	var config cfg.Config
	fmt.Println(dir)
	config.RootDir = dir
	if err := lockDataDirectory(&config); err != nil {
		t.Fatalf("Error: %v", err)
	}

	if err := lockDataDirectory(&config); err == nil {
		t.Fatalf("duplicate datadir failure mismatch: want %v", err)
	}
}
