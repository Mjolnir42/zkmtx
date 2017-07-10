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
# daemon start command
command: ...
# amount of time after which the running daemon is
# considered started
start.success.delay: 1m14s25µs
# what to do if the daemon dies
# - reaquire-lock, attempt to run the daemon again
# - run-command, run after.exit.failure commands
# - terminate, simply shut down
exit.policy: reaquire-lock|run-command|terminate
# commands run start.success.delay after the daemon
# started
after.start.success: [
  ...,
  ...
]
# commands run if the daemon exits != 0 and exit.policy
# is set to run-command
after.exit.failure: [
  ...,
  ...
]
# commands run always after the daemon exits
after.exit.always: [
  ...,
  ...
]
```

# Execute

```
/
├etc
│└── zkrun
│    ├── zkrun.conf
│    ├── zktest.conf
│    └── jobspec
│        └── enterprise_daemon.conf
└tmp
 └── testjob

# this will use the default /etc/zkrun/zkrun.conf as config and
# complete the jobspec to /etc/zkrun/jobspec/enterprise_daemon.conf
# since it is not an absolute path and does not end in .conf
zkrun --job enterprise_daemon

# this will also use enterprise_daemon.conf as jobspec
zkrun --job enterprise_daemon.conf

# this will use a different configuration file and use the jobspec as
# provided, since it is an absolute path
zkrun --config /etc/zkrun/zktest.conf --job /tmp/testjob
```
