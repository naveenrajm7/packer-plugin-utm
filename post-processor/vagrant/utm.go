package vagrant

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type UtmProvider struct{}

func (p *UtmProvider) KeepInputArtifact() bool {
	return false
}

func (p *UtmProvider) Process(ui packersdk.Ui, artifact packersdk.Artifact, dir string) (vagrantfile string, metadata map[string]interface{}, err error) {
	// Create the metadata
	metadata = map[string]interface{}{"provider": "utm"}

	// Identify the root .utm directory
	var utmDir string
	for _, path := range artifact.Files() {
		if strings.Contains(path, ".utm/") {
			utmDir = path[:strings.Index(path, ".utm/")+4]
			break
		}
	}

	if utmDir == "" {
		err = fmt.Errorf("no .utm directory found in artifact files")
		return
	}

	ui.Say(fmt.Sprintf("Copying .utm directory from artifact: %s", utmDir))
	dstPath := filepath.Join(dir, filepath.Base(utmDir))
	if err = CopyDirectoryContents(dstPath, utmDir); err != nil {
		return
	}

	// Rename the UTM file to box.utm, as required by Vagrant
	ui.Say("Renaming the UTM to box.utm...")
	utmDirPath := filepath.Join(dir, filepath.Base(utmDir))
	boxUtmPath := filepath.Join(dir, "box.utm")
	if err = os.Rename(utmDirPath, boxUtmPath); err != nil {
		return
	}

	// Use this to provide Vagrantfile with default values
	// vagrantfile = fmt.Sprintf(utmVagrantfile)
	vagrantfile = ""

	return
}
