package common

// PathValidationErrors contains constants for path validation error messages
const (
	// ErrBaseDirectoryNotSet Base directory errors
	ErrBaseDirectoryNotSet      = "base directory not set"
	ErrEmptyPath                = "path is empty"
	ErrPathContainsNullBytes    = "path contains null bytes"
	ErrPathExceedsMaxLength     = "path exceeds maximum length of %d characters"
	ErrFailedToResolveBasePath  = "failed to resolve base path: %w"
	ErrFailedToResolvePath      = "failed to resolve path: %w"
	ErrPathOutsideAllowedDir    = "path is outside of allowed directory: %s"
	ErrFailedToDetermineRelPath = "failed to determine relative path: %w"
	ErrPathTraversalDetected    = "path traversal attempt detected"

	// ErrFailedToEvaluateSymlink Symlink related errors
	ErrFailedToEvaluateSymlink  = "failed to evaluate symlink: %w"
	ErrFailedToEvalBaseDir      = "failed to evaluate base directory: %w"
	ErrFailedToResolveSymTarget = "failed to resolve symlink target path: %w"
	ErrFailedToResolveEvalBase  = "failed to resolve evaluated base path: %w"
	ErrSymlinkOutsideAllowedDir = "symlink points outside allowed directory: path is outside of allowed directory: %s"

	// ErrPathDoesNotExist File access errors
	ErrPathDoesNotExist   = "path does not exist: %s"
	ErrFailedToAccessPath = "failed to access path: %w"
	ErrNotRegularFile     = "not a regular file: %s"
)

// FileOperationErrors contains constants for file operation error messages
const (
	ErrInvalidFilePath       = "invalid file path: %w"
	ErrReadingFile           = "error reading file: %w"
	ErrCreatingDirectories   = "error creating directories: %w"
	ErrWritingTempFile       = "error writing temporary file: %w"
	ErrReplacingOriginalFile = "error replacing original file: %w"
	ErrOpeningSourceFile     = "error opening source file: %w"
	ErrCreatingDestFile      = "error creating destination file: %w"
	ErrCopyingFileContents   = "error copying file contents: %w"
	ErrSyncingFile           = "error syncing file: %w"
	ErrOpeningFileForAppend  = "error opening file for append: %w"
	ErrAppendingToFile       = "error appending to file: %w"
	ErrScanningDirectory     = "error scanning directory: %w"
)

// ScannerErrors contains constants for scanner error messages
const (
	ErrInvalidActionRefFormat  = "invalid action reference format: %s"
	ErrInvalidActionNameFormat = "invalid action name format: %s"
	ErrInvalidDirectoryPath    = "invalid directory path: %w"
	ErrWorkflowDirNotFound     = "workflows directory not found at %s"
	ErrScanningWorkflows       = "error scanning workflows: %w"
	ErrReadingWorkflowFile     = "error reading workflow file: %w"
	ErrParsingWorkflowYAML     = "error parsing workflow YAML: %w"
	ErrEmptyYAMLDocument       = "empty YAML document"
	ErrParsingWorkflowContent  = "error parsing workflow content: %w"
)

// TestErrors contains constants for test error messages - these maintain capitalization from the original test file
const (
	ErrFailedToRemoveTempDir  = "failed to remove temp directory: %v"
	ErrFailedToCreateTempDir  = "failed to create temp directory: %v"
	ErrFailedToCreateSubdir   = "failed to create subdirectory: %v"
	ErrFailedToCreateTestFile = "failed to create test file: %v"
	ErrFailedToRemoveSymlink  = "failed to remove symlink: %v"
	ErrFailedToGetWorkingDir  = "failed to get current working directory: %v"
	ErrFailedToChangeTempDir  = "failed to change to temporary directory: %v"
)

// VersionCheckerErrors contains constants for version checker error messages
const (
	ErrGettingTags         = "error getting tags: %w"
	ErrNoVersionInfo       = "no version information found for %s/%s"
	ErrGettingRefForTag    = "error getting ref for tag %s: %w"
	ErrNoCommitHashForTag  = "no commit hash found for tag %s"
	ErrGettingAnnotatedTag = "error getting annotated tag %s: %w"
	ErrNoCommitHashInTag   = "no commit hash found in annotated tag %s"
	ErrContextIsNil        = "context is nil"
)

// PRCreatorErrors contains constants for PR creator error messages
const (
	ErrCreatingBranch          = "error creating branch: %w"
	ErrCreatingCommit          = "error creating commit: %w"
	ErrCreatingPR              = "error creating pull request: %w"
	ErrGettingRepository       = "error getting repository: %w"
	ErrGettingDefaultBranchRef = "error getting default branch ref: %w"
	ErrGettingFileContents     = "error getting file contents: %w"
	ErrDecodingContent         = "error decoding content: %w"
	ErrCreatingBlob            = "error creating blob: %w"
	ErrGettingBranchRef        = "error getting branch ref: %w"
	ErrCreatingTree            = "error creating tree: %w"
)

// UpdateManagerErrors contains constants for update manager error messages
const (
	ErrInvalidUpdatePath = "invalid update path: %w"
	ErrReadingUpdateFile = "error reading file: %w"
	ErrWritingUpdateFile = "error writing file: %w"
	ErrApplyingUpdates   = "error applying updates: %w"
)

const (
	ErrAuthentication    = "authentication error: %w"
	ErrNetworkFailure    = "network failure: %w"
	ErrRateLimitExceeded = "GitHub API rate limit exceeded: %w"
	ErrNoRateLimitInfo   = "no rate limit information available"
	ErrRateLimitFormat   = "Rate limit: %d/%d, resets in %s"
)

const (
	ErrMissingRequiredFlag = "missing required flag: %s"
	ErrInvalidFlagValue    = "invalid value for flag %s: %s"
	ErrCommandExecution    = "error executing command: %w"
	ErrNoGithubToken       = "no GitHub token provided. Using public GitHub API with rate limiting. For higher rate limits, provide a token via -token flag or GITHUB_TOKEN environment variable." // #nosec G101
	ErrNoWorkflowsFound    = "no workflow files found"
	ErrNoUpdatesAvailable  = "no updates available"

	ErrFailedToCheckAction  = "failed to check %s/%s: %v"
	ErrFailedToCheckUpdate  = "failed to check update availability for %s/%s: %v"
	ErrFailedToCreateUpdate = "failed to create update for %s/%s: %v"
)

const (
	ErrInvalidTestParameters       = "invalid test parameters: %s"
	ErrWorkflowCountMustBePositive = "workflow count must be positive"
	ErrWorkflowCountExceedsLimit   = "workflow count exceeds maximum limit of %d"
	ErrCouldNotRemoveDummyFile     = "Warning: could not remove dummy file: %v"
	ErrFailedToParseWorkflow       = "failed to parse %s: %v"
)

const (
	ErrFailedToCloseBody = "failed to close response body: %v"
)

const (
	// ErrFailedToCreateRepoDir Repository operation failures
	ErrFailedToCreateRepoDir      = "failed to create repo directory: %v"
	ErrFailedToCreateWorkflowsDir = "failed to create workflows directory: %v"
	ErrFailedToChangePermissions  = "failed to make directory read-only: %v"
	ErrFailedToRestorePermissions = "failed to restore directory permissions: %v"
	ErrFailedToWriteWorkflowFile  = "failed to write invalid workflow file: %v"

	// ErrFailedToCloneRepo Git operation failures
	ErrFailedToCloneRepo     = "failed to clone repository: %v"
	ErrFailedToCommitChanges = "failed to commit changes: %v"
	ErrFailedToPushChanges   = "failed to push changes: %v"

	ErrFailedToCreateBranch = "failed to create branch: %v"
	ErrFailedToWriteFile    = "failed to write file: %v"

	ErrFailedToAddRemote    = "failed to add remote: %v"
	ErrFailedToSwitchBranch = "failed to switch branch: %v"

	// ErrExpectedError Validation failures
	ErrExpectedError   = "expected error %s, got nil"
	ErrUnexpectedError = "expected no error, got: %v"
	ErrExpectedResult  = "expected result %v, got %v"

	ErrExpectedPushError = "expected error when pushing to non-existent remote, got nil"

	ErrExpectedErrorContaining = "expected error containing %q, got %q"

	// ErrVersionCheckerNil Version checker test errors
	ErrVersionCheckerNil       = "NewDefaultVersionChecker() returned nil"
	ErrVersionCheckerClientNil = "NewDefaultVersionChecker() client is nil"
	ErrExpectedAuthClient      = "expected authenticated client, got unauthenticated"
	ErrExpectedUnauthClient    = "expected unauthenticated client, got authenticated"

	// ErrFailedToReadUpdatedFile File update test failures
	ErrFailedToReadUpdatedFile      = "failed to read updated file: %v"
	ErrExpectedContentNotFound      = "expected %q to be in the updated content, but it wasn't.\nUpdated content:\n%s"
	ErrExpectedNonExistentFileError = "expected error for non-existent file, got nil"
	ErrExpectedInvalidLineError     = "expected error for invalid line number, got nil"
	ErrFailedToRemoveTestFile       = "failed to remove test file: %v"
	ErrExpectedOutsidePathError     = "expected error for file outside base directory, got nil"
	ErrExpectedReadOnlyFileError    = "expected error for read-only file, but got nil. This might be system-dependent."
	ErrFailedToCreateEmptyFile      = "failed to create empty file: %v"
	ErrFailedToReadEmptyFile        = "failed to read empty file after update: %v"
	ErrExpectedVersionComment       = "expected empty file to contain version comment, got content: %s"
	ErrFailedToCreateSpecialFile    = "failed to create special file: %v"
	ErrFailedToReadSpecialFile      = "failed to read updated special file: %v"
	ErrFailedToCreateSameLineFile   = "failed to create same line file: %v"
	ErrFailedToReadSameLineFile     = "failed to read updated same line file: %v"

	ErrFailedToSetupTest           = "failed to set up test: %v"
	ErrExpectedActions             = "expected %d actions, got %d"
	ErrExpectedWorkflows           = "expected %d workflows, got %d"
	ErrExpectedEmptyCommitHash     = "expected empty commit hash for %s, got %q"
	ErrExpectedCommitHash          = "expected commit hash %s, got %q"
	ErrExpectedVersionFromComment  = "expected version %s from comment, got %q"
	ErrFailedToCreateTestFileNamed = "failed to create test file %s: %v"
	ErrSpecificWorkflowNotFound    = "%s not found"
	ErrUnexpectedWorkflowFile      = "unexpected workflow file: %s"
	ErrUnexpectedActionFound       = "unexpected action: %s/%s@%s"

	// ErrWorkDirNotCreated Additional test failures
	ErrWorkDirNotCreated             = "work directory not created: %v"
	ErrWrongDirectoryPermissions     = "wrong directory permissions: got %v, want %v"
	ErrWorkDirNotCleanedUp           = "work directory not cleaned up properly"
	ErrFailedToReadGitConfig         = "failed to read git config: %v"
	ErrGitConfigMissingValue         = "git config missing expected value: %s"
	ErrExpectedDifferentPaths        = "expected different repo paths for separate clones"
	ErrWorkflowFileNotFound          = "workflow file not found in %s"
	ErrFailedToStatFile              = "failed to stat file: %v"
	ErrWorkflowMissingContent        = "workflow file missing expected content: %s"
	ErrFailedToChangeFilePermissions = "failed to change file permissions: %v"
	ErrFailedToReadWorkflowFile      = "failed to read workflow file: %v"
	ErrWrongWorkflowContent          = "wrong workflow content: got %s, expected %s"
)
