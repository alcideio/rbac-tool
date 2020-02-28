
.SECONDARY:
.SECONDEXPANSION:


.phony: rbac-minimizer


get-deps: ##@Install Dependencies Linux
	go mod vendor


VERSION ?= 1.0.0

rbac-minimizer: ##@KubeDialer Build rbac-minimizer
	export CGO_ENABLED=0 && go build -o rbac-minimizer -tags staticbinary -i -v -ldflags='-s -w' .


HELP_FUN = \
         %help; \
         while(<>) { push @{$$help{$$2 // 'options'}}, [$$1, $$3] if /^(.+)\s*:.*\#\#(?:@(\w+))?\s(.*)$$/ }; \
         print "Usage: make [options] [target] ...\n\n"; \
     for (sort keys %help) { \
         print "$$_:\n"; \
         for (sort { $$a->[0] cmp $$b->[0] } @{$$help{$$_}}) { \
             $$sep = " " x (30 - length $$_->[0]); \
             print "  $$_->[0]$$sep$$_->[1]\n" ; \
         } print "\n"; }

help: ##@Misc Show this help
	@perl -e '$(HELP_FUN)' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

USERID=$(shell id -u)