# Packer Plugin Scaffolding

This repository is a template for a Packer multi-component plugin. It is intended as a starting point for creating Packer plugins, containing:
- A builder ([builder/scaffolding](builder/scaffolding))
- A provisioner ([provisioner/scaffolding](provisioner/scaffolding))
- A post-processor ([post-processor/scaffolding](post-processor/scaffolding))
- A data source ([datasource/scaffolding](datasource/scaffolding))
- Docs ([docs](docs))
- A working example ([example](example))

These folders contain boilerplate code that you will need to edit to create your own Packer multi-component plugin.
A full guide to creating Packer plugins can be found at [Extending Packer](https://www.packer.io/docs/plugins/creation).

In this repository you will also find a pre-defined GitHub Action configuration for the release workflow
(`.goreleaser.yml` and `.github/workflows/release.yml`). The release workflow configuration makes sure the GitHub
release artifacts are created with the correct binaries and naming conventions.

Please see the [GitHub template repository documentation](https://docs.github.com/en/free-pro-team@latest/github/creating-cloning-and-archiving-repositories/creating-a-repository-from-a-template)
for how to create a new repository from this template on GitHub.

## Packer plugin projects

Here's a non exaustive list of Packer plugins that you can checkout:

* [github.com/hashicorp/packer-plugin-docker](https://github.com/hashicorp/packer-plugin-docker)
* [github.com/exoscale/packer-plugin-exoscale](https://github.com/exoscale/packer-plugin-exoscale)
* [github.com/sylviamoss/packer-plugin-comment](https://github.com/sylviamoss/packer-plugin-comment)
* [github.com/hashicorp/packer-plugin-hashicups](https://github.com/hashicorp/packer-plugin-hashicups)

Looking at their code will give you good examples.

## Build from source

1. Clone this GitHub repository locally.

2. Run this command from the root directory: 
```shell 
go build -ldflags="-X github.com/hashicorp/packer-plugin-scaffolding/version.VersionPrerelease=dev" -o packer-plugin-scaffolding
```

3. After you successfully compile, the `packer-plugin-scaffolding` plugin binary file is in the root directory. 

4. To install the compiled plugin, run the following command 
```shell
packer plugins install --path packer-plugin-scaffolding github.com/hashicorp/scaffolding
```

### Build on *nix systems
Unix like systems with the make, sed, and grep commands installed can use the `make dev` to execute the build from source steps. 

### Build on Windows Powershell
The preferred solution for building on Windows are steps 2-4 listed above.
If you would prefer to script the building process you can use the following as a guide

```powershell
$MODULE_NAME = (Get-Content go.mod | Where-Object { $_ -match "^module"  }) -replace 'module ',''
$FQN = $MODULE_NAME -replace 'packer-plugin-',''
go build -ldflags="-X $MODULE_NAME/version.VersionPrerelease=dev" -o packer-plugin-scaffolding.exe
packer plugins install --path packer-plugin-scaffolding.exe $FQN
```

## Running Acceptance Tests

Make sure to install the plugin locally using the steps in [Build from source](#build-from-source).

Once everything needed is set up, run:
```
PACKER_ACC=1 go test -count 1 -v ./... -timeout=120m
```

This will run the acceptance tests for all plugins in this set.

## Registering Plugin as Packer Integration

Partner and community plugins can be hard to find if a user doesn't know what 
they are looking for. To assist with plugin discovery Packer offers an integration
portal at https://developer.hashicorp.com/packer/integrations to list known integrations 
that work with the latest release of Packer. 

Registering a plugin as an integration requires [metadata configuration](./metadata.hcl) within the plugin
repository and approval by the Packer team. To initiate the process of registering your 
plugin as a Packer integration refer to the [Developing Plugins](https://developer.hashicorp.com/packer/docs/plugins/creation#registering-plugins) page.

# Requirements

-	[packer-plugin-sdk](https://github.com/hashicorp/packer-plugin-sdk) >= v0.5.2
-	[Go](https://golang.org/doc/install) >= 1.20

## Packer Compatibility
This scaffolding template is compatible with Packer >= v1.10.2
