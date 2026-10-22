install:
	./scripts/install.sh

uninstall:
	./scripts/uninstall.sh

validate:
	./scripts/validate.sh

verify: test
	./scripts/verify.sh

test:
	python3 -m unittest discover -s tests -p 'test_native_*.py'
	bash ./tests/catalog_test.sh
	bash ./tests/validation_test.sh
	./tests/installer_test.sh
	bash ./tests/selection_test.sh
	bash ./tests/managed_state_test.sh
	bash ./tests/managed_reconcile_test.sh
	bash ./tests/managed_guidance_test.sh
