install:
	./scripts/install.sh

uninstall:
	./scripts/uninstall.sh

validate:
	./scripts/validate.sh

verify: test
	./scripts/verify.sh

test:
	bash ./tests/validation_test.sh
	./tests/installer_test.sh
	bash ./tests/selection_test.sh
