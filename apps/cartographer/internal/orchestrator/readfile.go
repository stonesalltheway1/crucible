package orchestrator

import "os"

func _readFile(p string) ([]byte, error) {
	return os.ReadFile(p)
}
