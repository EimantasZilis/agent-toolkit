# Historical native fixtures

`historical-release.zip` contains a release built by the Python implementation
from the skill catalog at source commit `f408f60`, using version `0.1.0`.
The outer ZIP stores the complete release tree and executable modes. Its inner
provider ZIPs retain the historical archive format and checksums.

Tests unpack this fixture in a temporary directory and validate it without the
generator or its source checkout. The fixture is data; tests never execute its
bundled resources.

`historical-snapshot.zip` contains the combined snapshot produced by the Python
snapshot test fixture using the same catalog. Its provenance identifies the
fixture's isolated committed source tree. Tests preserve its original schema,
inventories, and modes and validate it without that source tree.
