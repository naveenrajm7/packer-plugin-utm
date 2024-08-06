package zip

import (
	"fmt"
	"os"
)

const BuilderId = "naveenrajm7.utm.post-processor.zip"

// Artifact is the result of running the UTM zip post-processor,
// namely a zip file which contains a utm directory (UTM VM bundle).
type Artifact struct {
	Path string
}

func (a *Artifact) BuilderId() string {
	return BuilderId
}

func (*Artifact) Id() string {
	return ""
}

func (a *Artifact) Files() []string {
	return []string{a.Path}
}

func (a *Artifact) String() string {
	return fmt.Sprintf("compressed artifacts in: %s", a.Path)
}

func (*Artifact) State(name string) interface{} {
	return nil
}

func (a *Artifact) Destroy() error {
	return os.Remove(a.Path)
}
