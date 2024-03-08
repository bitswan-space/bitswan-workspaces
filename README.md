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
   * [Makefile Targets](#makefile-targets)
   * [Contribute](#contribute)

<!-- Added by: morelly_t1, at: Tue 10 Aug 2021 08:54:24 AM CEST -->

<!--te-->

# Features
- Automatically set up independent bitswan-gitops deployments.
- Deployments can either connect to the bitswan.space SaaS, use the on prem bitswan management tools or operate completely independently

# Makefile Targets
```sh
$> make
bootstrap                      install build deps
build                          build golang binary
clean                          clean up environment
cover                          display test coverage
docker-build                   dockerize golang application
fmt                            format go files
help                           list makefile targets
install                        install golang binary
lint                           lint go files
pre-commit                     run pre-commit hooks
run                            run the app
test                           display test coverage
```

# Contribute
If you find issues in that setup or have some nice features / improvements, I would welcome an issue or a PR :)
