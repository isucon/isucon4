# ISUCON4 Qualifier Benchmarker

## Prerequisite

- Gondler
  - `gem install gondler`

## Build

### Debug

```
$ make debug
```

Debug build is:

- Verbose; shows request log, etc
- no require of AWS EC2 environment (`http://169.254.169.254/latest/meta-data/*`)
- Default value of `init.sh` is `./init.sh`
- Find tsv files from `./sql`
  - Both depends on cwd

This build shouldn't be included in AMI.

### Release

```
$ make release
```

Release build is:

- requires to run on AWS EC2 environment.
- Default value of `init.sh` is `/home/isucon/init.sh`
- Find tsv files from `/home/isucon/sql`

This build should be included in AMI.

## Run

- Make sure TSV files is available on correct path. (debug: `./sql`, release: `/home/isucon/sql`)
