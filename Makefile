TARGETS := $(shell ls scripts)


$(TARGETS): 
	dapper $@

trash: 
	dapper -m bind trash

trash-keep: 
	dapper -m bind trash -k

deps: trash

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)
