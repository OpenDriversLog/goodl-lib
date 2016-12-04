build-test:
	@rm -rf Dockerfile
	@cp ./dockerize/Dockerfile ./Dockerfile
	docker build --tag=odl_go/lib_test .

run-bash:
	docker run -it --rm \
		odl_go/lib_test bash

test-all: test-dbMan test-datapolish test-addressManager test-tripMan

test-dbMan:
	docker run -it --rm \
		odl_go/lib_test ginkgo --keepGoing --noisyPendings=false \
	/go/src/github.com/OpenDriversLog/goodl-lib/dbMan

test-datapolish:
	docker run -it --rm \
		odl_go/lib_test ginkgo --keepGoing --noisyPendings=false \
	/go/src/github.com/OpenDriversLog/goodl-lib/datapolish

test-addressManager:
	docker run -it --rm \
		odl_go/lib_test ginkgo --keepGoing --noisyPendings=false \
	/go/src/github.com/OpenDriversLog/goodl-lib/jsonapi/addressManager

test-tripMan:
	docker run -it --rm \
		odl_go/lib_test ginkgo --keepGoing --noisyPendings=false \
	/go/src/github.com/OpenDriversLog/goodl-lib/jsonapi/tripMan

