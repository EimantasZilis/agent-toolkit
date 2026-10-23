install:
	./scripts/install.sh

uninstall:
	./scripts/uninstall.sh

validate:
	./scripts/validate.sh

verify: test
	./scripts/verify.sh

test:
	python3 -m unittest discover -s tests -p 'test_tui_*.py'
	python3 -m unittest discover -s tests -p 'test_native_*.py'
	python3 -m unittest discover -s tests -p 'test_managed_*.py'
	bash ./tests/catalog_test.sh
	bash ./tests/validation_test.sh
	./tests/installer_test.sh
	bash ./tests/selection_test.sh
	bash ./tests/managed_state_test.sh
	bash ./tests/managed_reconcile_test.sh
	bash ./tests/managed_guidance_test.sh

# Example: make release-tui VERSION=v0.1.0
release-tui:
	python3 scripts/build_tui.py --version "$(VERSION)" --output "dist/tui/$(VERSION)"
