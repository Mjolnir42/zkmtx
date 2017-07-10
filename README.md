# WORK IN PROGRESS

# zkrun

Zookeeper Run - run a daemon under Zookeeper lock.

`zkrun` tries to aquire its distributed lock and runs the job
specification if successful.

# Zookeeper layout

```
/
└── zkrun
    └── <syncgroup>
        └── <job>
            ├── active
            └── lock
```

# Configuration

```
/etc
 └── zkrun
     ├── zkrun.conf
     └── jobspec
         ├── ...
         └── foobar.conf

/etc/zkrun/zkrun.conf:
ensemble: <zk-connect-string>
syncgroup: <name>

/etc/zkrun/jobspec/foobar.conf:
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
zkrun -lock <name> -- ${cmd}
```

go doc syscall.SysProcAttr
go doc syscall.Credential
