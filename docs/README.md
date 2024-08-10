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
$ packer plugins install github.com/naveenrajm7/utm
```

### Components

The plugin comes with a builder and a post-processor to create UTM
machines.
The following UTM Builders and post-processors are supported.

#### Builders

- [utm-utm](builders/utm.mdx) - This builder imports
  an existing UTM file, runs provisioners on top of that VM, and exports
  that machine to create an image (.utm). This is best if you have an existing
  UTM VM export you want to use as the source. As an additional
  benefit, you can feed the artifact of this builder back into itself to
  iterate on a machine.

#### Post-processors

- [utm-zip](post-processors/zip.mdx) - The utm zip post-processor is 
simplied version of compress zip post-processor. This post-processor takes 
in the artifact from UTM builders and zips up the UTM directory, which
can be used to share and import VMs in UTM.
You can use the zip version of UTM VM either through [Vagrant UTM plugin](https://github.com/naveenrajm7/vagrant_utm) or directly through [`downloadVM?url=...`](https://docs.getutm.app/advanced/remote-control/)
