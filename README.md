# Golang backend Library for ODL
See https://github.com/OpenDriversLog/goodl
# License 
This work is licensed under a [![License](https://i.creativecommons.org/l/by-nc-sa/4.0/80x15.png) Creative Commons Attribution-NonCommercial-ShareAlike 4.0 International License](https://creativecommons.org/licenses/by-nc-sa/4.0/).
To view a copy of this license, visit http://creativecommons.org/licenses/by-nc-sa/4.0/ or send a letter to Creative Commons, PO Box 1866, Mountain View, CA 94042, USA.
# Makefile & Dockerfile magic

## Content

- dbManager with migration support
- database call handling
- addressManager
- data-processing functions ,etc
- debug functions
- json-response API for DB stuff

## Usage

__IMPORTANT__ all non-core dependencies must be listed in the `glide.yaml` file to build successfully!

CI tests & deploy using a docker container... Jenkins (not active anymore) did that after you created a Merge Request.
Usually you want to run the tests locally using

1. `make build-test`
2. `make test-all`

## future plans

- cross platform
- 1 library on web & Android & iOS
