# Maintainer Instructions

- Always preserve backward compatibility
- Build using `make clean && make`
- After merging PR, alway run `make changelog` and commit changes
- Set ArangoDB docker container (used for testing) using `export ARANGODB=<image-name>`
- Run tests using:
  - `make run-tests-single`
  - `make run-tests-resilientsingle`
  - `make run-tests-cluster`.
- Always create changes in a PR
