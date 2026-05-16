package api

import "os"

func lookupEnvImpl(name string) (string, bool) {
	return os.LookupEnv(name)
}
