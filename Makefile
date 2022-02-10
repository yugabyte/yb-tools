include global.mk

YBTOOLS = pkg ycrc yugatool yugaware-client

all: ${YBTOOLS}

.PHONY: ${YBTOOLS}
${YBTOOLS}:
	${MAKE} -C $@

clean:
	rm -rf bin/
	rm -rf out/

.PHONY: integration
integration: ginkgo
	ginkgo test ./integration/...
