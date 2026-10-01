install:
	./scripts/install.sh

uninstall:
	./scripts/uninstall.sh

validate:
	./scripts/validate.sh

verify: test
	./scripts/verify.sh

test:
	./tests/installer_test.sh
