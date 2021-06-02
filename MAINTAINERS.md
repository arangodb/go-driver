# Maintainer Instructions

- Always preserve backward compatibility
- Build using `make clean && make`
- After merging PR, alway run `make changelog` and commit changes
- Set ArangoDB docker container (used for testing) using `export ARANGODB=<image-name>`
- Run tests using:
  - `make run-tests-single`
  - `make run-tests-resilientsingle`
  - `make run-tests-cluster`.
- The test can be launched with the flag `RACE=on` which means that test will be performed with the race detector, e.g:
  - `RACE=on make run-tests-single`
- Always create changes in a PR
