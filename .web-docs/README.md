The UTM plugin is able to create
[UTM](https://mac.getutm.app/) virtual machines and export them in
the .utm format.

### Installation

To install this plugin, copy and paste this code into your Packer configuration, then run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    utm = {
      version = ">=v0.0.2"
      source  = "github.com/naveenrajm7/utm"
    }
  }
}
```

Alternatively, you can use `packer plugins install` to manage installation of this plugin.

```sh
packer plugins install github.com/naveenrajm7/utm
```

### Components

The plugin comes with multiple builders and post-processors to create UTM machines.
The following UTM Builders and post-processors are supported.

#### Builders

- [utm-iso](builders/iso.mdx) - Starts from an ISO file, creates a brand-new UTM VM, installs an OS, provisions software within the OS, then exports that machine to create an image.
This is best for people who want to start from scratch.

- [utm-cloud](builders/cloud.mdx) - This builder imports
  an existing qcow2 file with cloud-init (cloud image),
  feeds in your custom cloud-init seed file,
  runs provisioners on top of that VM,
  and exports that machine to create an UTM image (.utm).
  This is best for people who want to start off or test with cloud images,
  which are provided by most popular distros.

- [utm-utm](builders/utm.mdx) - This builder uses an existing UTM VM to run defined provisioners on top of that VM.
  This is best if you have an existing
  UTM VM export you want to use as the source.
  As an additional benefit,
  you can feed the artifact of this builder back into itself to
  iterate on a machine.

#### Post-processors

- [utm-zip](post-processors/zip.mdx) - The utm zip post-processor is
simplied version of The Packer compress zip post-processor modified to accept utm directory. This post-processor takes
in the artifact from UTM builders and zips up the UTM directory, which
can be used to share and import VMs in UTM.
You can use the zip version of UTM VM in UTM through [`downloadVM?url=...`](https://docs.getutm.app/advanced/remote-control/)

- [utm-vagrant](post-processors/vagrant.mdx) - The UTM Vagrant post-processor is a modified version of The Packer Vagrant post-processor to accommodate utm directory.
This takes a build and converts the artifact into a valid Vagrant box. The artifact of this post-processor can be feed into the 'artifice' post-processor and later into vagrant-registry post-processor to publish your UTM vagrant boxes to HCP Vagrant Box Registry.
