package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/hibooboo2/ggit/giturls"
	"github.com/muja/goconfig"
	"github.com/pkg/errors"
)

func main() {
	v := flag.Bool("verbose", false, "Verbose logging")
	flag.Parse()
	if !*v {
		log.SetOutput(ioutil.Discard)
	}
	log.SetFlags(log.Llongfile)
	r, err := LoadRepo("")
	if err != nil {
		log.Printf("Failed to load repo: %+v", err)
		os.Exit(42)
	}
	command := "info"
	if len(os.Args) > 1 {
		command = strings.ToLower(os.Args[1])
	}
	switch command {
	case "info":
		log.Println(*r)
	case "ticket":
		var branch string
		if len(os.Args) < 3 {
			data, err := ioutil.ReadFile(".git/HEAD")
			if err != nil {
				panic(err)
			}
			if string(data)[:4] == "ref:" {
				branch = strings.TrimSpace(strings.TrimPrefix(string(data), "ref: refs/heads/"))
				log.Println("Branch is:", branch)
			}
		} else {
			branch = os.Args[2]
		}
		r.openTicketForBranch(branch)
	default:
		log.Printf("Unknown command [%s]", command)
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
	r := &Repo{Remotes: map[string]Remote{}, Branches: map[string]Branch{}}
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
		case "branch":
			branchName := vals[1]
			_, ok := r.Branches[branchName]
			if !ok {
				branch := Branch{Name: branchName, Remote: config[fmt.Sprintf("branch.%s.remote", branchName)], Merge: config[fmt.Sprintf("branch.%s.merge")]}
				r.Branches[branchName] = branch
			}
		}
	}
	info, err := ioutil.ReadDir(".git/refs/heads")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get info for heads")
	}
	for _, inf := range info {
		if inf.IsDir() {
			continue
		}
		branch := inf.Name()
		if strings.ToUpper(branch) == "HEAD" {
			continue
		}
		_, ok := r.Branches[branch]
		if !ok {
			r.Branches[branch] = Branch{Name: branch}
		}
	}
	return r, nil
}

type Repo struct {
	Remotes  map[string]Remote
	Branches map[string]Branch
}

type Branch struct {
	Name   string
	Remote string
	Merge  string
}

type Remote struct {
	Url   string
	Fetch string
}

func (r *Repo) openTicketForBranch(branch string) {
	b, ok := r.Branches[branch]
	if !ok {
		log.Printf("Branch %s not found", branch)
		return
	}
	remoteType := r.getRemoteType(b.Remote)
	if remoteType == Unknown {
		log.Printf("Do not know how to open ticket for branch %s it may not be formed in a way to get a ticket for it or it has not remote", branch)
		return
	}
	if remoteType == Notset {
		log.Println("Remote not set assuming its for acronis as that is what this was mostly written for")
		remoteType = Acronis
	}
	log.Printf("Branch %s remote type is %s", branch, remoteType)
	reg, ok := remoteTypeTicketRegexes[remoteType]
	if !ok {
		log.Printf("No regexp for remote type %s found for branch %s", remoteType, branch)
		return
	}
	if !reg.MatchString(branch) {
		log.Printf("Branch %s does not have an associated ticket or is not formed correctly", branch)
		return
	}
	loc := reg.FindStringSubmatchIndex(branch)
	log.Println(branch[loc[0]:loc[1]])
	_, err := exec.Command("open", fmt.Sprintf(remoteTypeTicketFormatString[remoteType], branch)).CombinedOutput()
	if err != nil {
		panic(err)
	}
}

func (r *Repo) getRemoteType(remote string) string {
	if remote == "" {
		return Notset
	}
	u, err := giturls.Parse(r.Remotes[remote].Url)
	if err != nil {
		log.Println(err)
		return Unknown
	}

	remoteType, ok := remoteTypes[u.Hostname()]
	if !ok {
		log.Println(u.Host)
		return Unknown
	}
	return remoteType
}

const (
	Acronis = "acronis"
	Github  = "github"
	Unknown = "unknown"
	Notset  = "notset"
)

var remoteTypes = map[string]string{
	"git.acronis.com": Acronis,
	"github.com":      Github,
}

var remoteTypeTicketRegexes = map[string]*regexp.Regexp{
	Acronis: regexp.MustCompile(`(ABR-\d{6})`),
}

var remoteTypeTicketFormatString = map[string]string{
	Acronis: "https://pmc.acronis.com/browse/%s",
}
