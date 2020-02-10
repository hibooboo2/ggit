package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/muja/goconfig"
	"github.com/pkg/errors"
)

func main() {
	r, err := LoadRepo("")
	if err != nil {
		log.Printf("Failed to load repo: %+v", err)
		os.Exit(42)
	}
	for _, remote := range r.Remotes {
		log.Println(remote)
	}
}

func LoadRepo(location string) (*Repo, error) {
	if location == "" {
		location = "."
	}

	inf, err := os.Open(path.Join(location, ".git/config"))
	if os.IsNotExist(err) {
		return nil, errors.New("Location is not a valid git repository")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to open git config")
	}

	data, err := ioutil.ReadAll(inf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read git config")
	}

	config, _, err := goconfig.Parse(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse git config")
	}
	r := &Repo{Remotes: map[string]Remote{}}
	for k := range config {
		vals := strings.Split(k, ".")
		switch vals[0] {
		case "remote":
			remoteName := vals[1]
			_, ok := r.Remotes[remoteName]
			if !ok {
				remote := Remote{Url: config[fmt.Sprintf("remote.%s.url", remoteName)], Fetch: config[fmt.Sprintf("remote.%s.fetch", remoteName)]}
				r.Remotes[remoteName] = remote
			}
		}
	}
	return r, nil
}

type Repo struct {
	Remotes map[string]Remote
}

type Remote struct {
	Url   string
	Fetch string
}

func (r *Repo) openTicketForBranch(branch string) {

}
