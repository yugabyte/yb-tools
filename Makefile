include global.mk

YBTOOLS = pkg ycrc yugatool yugaware-client yb-support-tool

all: ${YBTOOLS}

.PHONY: ${YBTOOLS}
${YBTOOLS}:
	${MAKE} -C $@


.PHONY: integration
integration: ginkgo
	ginkgo test ./integration/...
