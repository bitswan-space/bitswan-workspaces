# bitswan-gitops

<div align="center">
CLI app for managing bitswan-gitops deployments
<br>
<br>
<img src="https://github.com/bitswan-space/bitswan-gitops/actions/workflows/test.yml/badge.svg" alt="drawing"/>
<img src="https://github.com/bitswan-space/bitswan-gitops/actions/workflows/lint.yml/badge.svg" alt="drawing"/>
<img src="https://pkg.go.dev/badge/github.com/bitswan-space/bitswan-gitops.svg" alt="drawing"/>
<img src="https://codecov.io/gh/bitswan-space/bitswan-gitops/branch/main/graph/badge.svg" alt="drawing"/>
<img src="https://img.shields.io/github/v/release/bitswan-space/bitswan-gitops" alt="drawing"/>
<img src="https://img.shields.io/docker/pulls/bitswan-space/bitswan-gitops" alt="drawing"/>
<img src="https://img.shields.io/github/downloads/bitswan-space/bitswan-gitops/total.svg" alt="drawing"/>
</div>

# Table of Contents
<!--ts-->
   * [bitswan-gitops](#bitswan-gitops)
   * [Features](#features)
   * [Contribute](#contribute)

<!--te-->

# Features
- Automatically set up independent bitswan-gitops deployments.
- Deployments can either connect to the bitswan.space SaaS, use the on prem bitswan management tools or operate completely independently


# Installation

```
# Download and extract the binary in one command
curl -L https://github.com/bitswan-space/bitswan-gitops-cli/releases/download/0.1.6/bitswan-gitops-cli_Linux_x86_64.tar.gz | tar -xz
```

Move the binary to a directory in your PATH

```
sudo mv bitswan-gitops-cli /usr/local/bin/
```

Alternatively, if you don't have sudo access or prefer a local installation:

```
mkdir -p ~/bin
mv bitswan-gitops-cli ~/bin/

#Add to PATH if using ~/bin (add this to your ~/.bashrc or ~/.zshrc)
export PATH="$HOME/bin:$PATH"
```

# Setting up and connecting a gitops instance
## SaaS
```sh
bitswan-gitops-cli init my-gitops
```

## On-prem
### With public domain
```sh
bitswan-gitops-cli init --domain=my-gitops.bitswan.io my-gitops
```
### With internal domain
> Note:
>
> Before you initialize gitops with internal domain, make sure you have generated certificate for sub domain of gitops instance, e.g. `*.my-gitops.my-domain.local`. You have to specify path to the certificate and private key in `init` command. Certificate and private key must be in a format `full-chain.pem` and `private-key.pem`.

```sh
bitswan-gitops-cli init --domain=my-gitops.my-domain.local --certs-dir=/etc/certs my-gitops
```

### Local dev

This is for setting up gitops locally without first setting up a domain name or connecting to the SaaS.

First add to /etc/hosts:

```sh
      127.0.0.1 editor.bitswan.localhost
      127.0.0.1 gitops.bitswan.localhost
      127.0.0.1 testpipeline.bitswan.localhost
```

Create some certs for these domains using a certificate authority you setup for yourself. 

```sh
mkdir bitswan-certs
cd bitswan-certs
```

```sh
mkcert --install
```

Add the CA certificate to Chrome by:
1. Navigate to chrome://settings/certificates
2. Go to "Authorities" tab
3. Click "Import" and select the ca.crt file
4. Check all trust settings and click "OK"

Now generate the wildcard certificates:
```
mkcert "*.bitswan.localhost"

mv _wildcard.bitswan.localhost-key.pem private-key.pem
mv _wildcard.bitswan.localhost.pem full-chain.pem

```

And finally setup the gitops.

```sh
bitswan-gitops-cli init --domain=bitswan.localhost --certs-dir=$PWD dev-gitops
```

You should be able to access the editor in chrome via https://editor.localhost.

You can get the password to the editor using the command:

```sh
docker exec -it dev-gitops-site-bitswan-editor-dev-gitops-1 cat /home/coder/.config/code-server/config.yaml
```



## Remote git repository
If you wanna connect and persist your pipelines and GitOps configuration in remote git repository you can use `--remote` flag to specify your repository. `main` branch will be used to store pipelines code and each GitOps will create it's own branch (e.g. `my-gitops`) to store their configurations.

```sh
bitswan-gitops-cli --remote=git@github.com:<your-name>/<your-repo>.git my-gitops
```

# Contribute
If you find issues in that setup or have some nice features / improvements, I would welcome an issue or a PR :)
