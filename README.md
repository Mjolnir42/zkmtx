# WORK IN PROGRESS

# zkmtx

Zookeeper Mutex - ensure a command is running once.

`zkmtx` tries to aquire its distributed lock and runs the job
specification if successful.

# Zookeeper layout

```
/
└── zkmtx
    └── <syncgroup>
        └── <job>
            ├── active
            └── lock
```

# Configuration

```
/etc
 └── zkmtx
     ├── zkmtx.conf
     └── jobspec
         ├── ...
         └── foobar.conf

/etc/zkmtx/zkmtx.conf:
ensemble: <zk-connect-string>
syncgroup: <name>

/etc/zkmtx/jobspec/foobar.conf:
command: ...
exit.policy: reaquire-lock|run-command|terminate
after.start.success: [
  ...,
  ...
]
after.exit.failure: [
  ...,
  ...
]
after.exit.always: [
  ...,
  ...
]
```

# Execute

```
zkmtx -lock <name> -- ${cmd}
```

go doc syscall.SysProcAttr
go doc syscall.Credential
