// This package is used to store the version of the server during runtime.
// The values are set during runtime in the main package.
package version

var (
	// Version is the version of the server.
	Version = ""

	// CommitSHA is the commit SHA of the server.
	CommitSHA = ""

	// CommitDate is the commit date of the server.
	CommitDate = ""
)
