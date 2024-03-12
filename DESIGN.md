Commands
---------

- `bitswan-gitops clone --bitswan-dir=<bitswan-dir> <repo> <dest>`
- `bitswan-gitops start-ide <deployment-id>`
- `bitswan-gitops pull`
- `bitswan-gitops admin-connect`

clone
------

- `bitswan-gitops clone --bitswan-dir=<bitswan-dir> <repo> <dest>`

Clone with create a directory named `dest` and then clone the git repo into a subdirectory named `prod`. It will also copy `prod` to `dev`. The prod directory is the directory from which gitops will launch services. The dev directory is a directory that launched IDEs can use to edit the source of services.

The data flow (sourcecode flow) is similar to this diagram.

```
                 -----------------
  -------------->| github/gitlab | ---------------------
  |              -----------------                     |
  |                                                    |
  | -------------------------------------------------- |
  | |        CRE (Virtual Machine or K8S cluster)    | |
  | |                                                | |
  | |     --------------         ---------------     | |
  --------| Development|         | Production  |<-------
    |     |  git repo  |         | git repo    |     |
    |     --------------         ---------------     |
    |           ↑                      ↓             |
    |     --------------         ---------------     |
    |     | JupyterLab |         | Pipelines   |     |
    |     |   Web IDE  |         |             |     |
    |     --------------         ---------------     |
    |                                                |
    --------------------------------------------------
```

start-ide
-----------

- `bitswan-gitops start-ide <deployment-id>`

The start ide command launches the IDE associated with the given deployment. The URL of the IDE will be printed to stdout.

pull
-----
- `bitswan-gitops start-ide`

Pull any changes from the remote to the production git repository and critically, also rebuild and redeploy any pipelines/IDEs that are effected by the changes.

admin-connect
----------------

Connect to the bitswan.space SaaS service to view and manage your pipelines.
