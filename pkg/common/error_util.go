package common

// PathValidationErrors contains constants for path validation error messages
const (
	// Base directory errors
	ErrBaseDirectoryNotSet      = "base directory not set"
	ErrEmptyPath                = "path is empty"
	ErrPathContainsNullBytes    = "path contains null bytes"
	ErrPathExceedsMaxLength     = "path exceeds maximum length of %d characters"
	ErrFailedToResolveBasePath  = "failed to resolve base path: %w"
	ErrFailedToResolvePath      = "failed to resolve path: %w"
	ErrPathOutsideAllowedDir    = "path is outside of allowed directory: %s"
	ErrFailedToDetermineRelPath = "failed to determine relative path: %w"
	ErrPathTraversalDetected    = "path traversal attempt detected"

	// Symlink related errors
	ErrFailedToEvaluateSymlink  = "failed to evaluate symlink: %w"
	ErrFailedToEvalBaseDir      = "failed to evaluate base directory: %w"
	ErrFailedToResolveSymTarget = "failed to resolve symlink target path: %w"
	ErrFailedToResolveEvalBase  = "failed to resolve evaluated base path: %w"
	ErrSymlinkOutsideAllowedDir = "symlink points outside allowed directory: path is outside of allowed directory: %s"

	// File access errors
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
	ErrFailedToRemoveTempDir  = "Failed to remove temp directory: %v"
	ErrFailedToCreateTempDir  = "Failed to create temp directory: %v"
	ErrFailedToCreateSubdir   = "Failed to create subdirectory: %v"
	ErrFailedToCreateTestFile = "Failed to create test file: %v"
	ErrFailedToRemoveSymlink  = "Failed to remove symlink: %v"
	ErrFailedToGetWorkingDir  = "Failed to get current working directory: %v"
	ErrFailedToChangeTempDir  = "Failed to change to temporary directory: %v"
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

// GitHubErrors contains constants for GitHub utility error messages
const (
	ErrCreatingGitHubClient = "error creating GitHub client: %w"
	ErrAuthentication       = "authentication error: %w"
	ErrNetworkFailure       = "network failure: %w"
	ErrRateLimitExceeded    = "GitHub API rate limit exceeded: %w"
	ErrNoRateLimitInfo      = "No rate limit information available"
	ErrRateLimitFormat      = "Rate limit: %d/%d, resets in %s"
	ErrInvalidEnterpriseURL = "invalid enterprise URL: %w"
)

// CommandErrors contains constants for command line errors
const (
	ErrMissingRequiredFlag   = "missing required flag: %s"
	ErrInvalidFlagValue      = "invalid value for flag %s: %s"
	ErrCommandExecution      = "error executing command: %w"
	ErrNoGithubToken         = "No GitHub token provided. Using public GitHub API with rate limiting. For higher rate limits, provide a token via -token flag or GITHUB_TOKEN environment variable."
	ErrNoWorkflowsFound      = "No workflow files found"
	ErrNoUpdatesAvailable    = "No updates available"
	ErrFailedToParseWorkflow = "Failed to parse %s: %v"
	ErrFailedToCheckAction   = "Failed to check %s/%s: %v"
	ErrFailedToCheckUpdate   = "Failed to check update availability for %s/%s: %v"
	ErrFailedToCreateUpdate  = "Failed to create update for %s/%s: %v"
)

// TestToolErrors contains constants for test tool error messages
const (
	ErrGeneratingTestData          = "error generating test data: %w"
	ErrInvalidTestParameters       = "invalid test parameters: %s"
	ErrWorkflowCountMustBePositive = "Workflow count must be positive"
	ErrWorkflowCountExceedsLimit   = "Workflow count exceeds maximum limit of %d"
	ErrCouldNotRemoveDummyFile     = "Warning: could not remove dummy file: %v"
)

// TestFailureErrors contains constants for test failure messages
const (
	// Repository operation failures
	ErrFailedToCreateRepoDir      = "Failed to create repo directory: %v"
	ErrFailedToCreateWorkflowsDir = "Failed to create workflows directory: %v"
	ErrFailedToChangePermissions  = "Failed to make directory read-only: %v"
	ErrFailedToRestorePermissions = "Failed to restore directory permissions: %v"
	ErrFailedToWriteWorkflowFile  = "Failed to write invalid workflow file: %v"

	// Git operation failures
	ErrFailedToCloneRepo        = "Failed to clone repository: %v"
	ErrFailedToCommitChanges    = "Failed to commit changes: %v"
	ErrFailedToPushChanges      = "Failed to push changes: %v"
	ErrFailedToCorruptGitConfig = "Failed to corrupt git config: %v"
	ErrFailedToCreateBranch     = "Failed to create branch: %v"
	ErrFailedToWriteFile        = "Failed to write file: %v"
	ErrFailedToStageChanges     = "Failed to stage changes: %v"
	ErrFailedToAddRemote        = "Failed to add remote: %v"
	ErrFailedToSwitchBranch     = "Failed to switch branch: %v"

	// Setup/Cleanup failures
	ErrFailedToSetupTestEnv   = "Failed to set up test environment: %v"
	ErrFailedToCleanupTestEnv = "Failed to clean up test environment: %v"

	// Validation failures
	ErrExpectedError           = "Expected error %s, got nil"
	ErrUnexpectedError         = "Expected no error, got: %v"
	ErrExpectedResult          = "Expected result %v, got %v"
	ErrExpectedGitConfigError  = "Expected error with corrupted git config, got nil"
	ErrExpectedPushError       = "Expected error when pushing to non-existent remote, got nil"
	ErrUnexpectedErrorMessage  = "Unexpected error message: %s"
	ErrExpectedCommitError     = "Expected error when committing without staged changes, got nil"
	ErrExpectedBranchError     = "Expected error when creating branch with invalid name, got nil"
	ErrExpectedMergeError      = "Expected error when merging conflicting branches, got nil"
	ErrExpectedErrorContaining = "Expected error containing %q, got %q"

	// Version checker test errors
	ErrVersionCheckerNil       = "NewDefaultVersionChecker() returned nil"
	ErrVersionCheckerClientNil = "NewDefaultVersionChecker() client is nil"
	ErrExpectedAuthClient      = "Expected authenticated client, got unauthenticated"
	ErrExpectedUnauthClient    = "Expected unauthenticated client, got authenticated"

	// File update test failures
	ErrFailedToReadUpdatedFile      = "Failed to read updated file: %v"
	ErrExpectedContentNotFound      = "Expected %q to be in the updated content, but it wasn't.\nUpdated content:\n%s"
	ErrExpectedNonExistentFileError = "Expected error for non-existent file, got nil"
	ErrExpectedInvalidLineError     = "Expected error for invalid line number, got nil"
	ErrFailedToRemoveTestFile       = "Failed to remove test file: %v"
	ErrExpectedOutsidePathError     = "Expected error for file outside base directory, got nil"
	ErrExpectedReadOnlyFileError    = "Expected error for read-only file, but got nil. This might be system-dependent."
	ErrFailedToCreateEmptyFile      = "Failed to create empty file: %v"
	ErrFailedToReadEmptyFile        = "Failed to read empty file after update: %v"
	ErrExpectedVersionComment       = "Expected empty file to contain version comment, got content: %s"
	ErrFailedToCreateSpecialFile    = "Failed to create special file: %v"
	ErrFailedToReadSpecialFile      = "Failed to read updated special file: %v"
	ErrFailedToCreateSameLineFile   = "Failed to create same line file: %v"
	ErrFailedToReadSameLineFile     = "Failed to read updated same line file: %v"

	// Scanner test errors
	ErrFailedToSetTempDirPermissions = "Failed to set temp dir permissions: %v"
	ErrFailedToSetupTest             = "Failed to set up test: %v"
	ErrExpectedActions               = "Expected %d actions, got %d"
	ErrExpectedWorkflows             = "Expected %d workflows, got %d"
	ErrExpectedEmptyCommitHash       = "Expected empty commit hash for %s, got %q"
	ErrExpectedCommitHash            = "Expected commit hash %s, got %q"
	ErrExpectedVersionFromComment    = "Expected version %s from comment, got %q"
	ErrFailedToCreateTestFileNamed   = "Failed to create test file %s: %v"
	ErrSpecificWorkflowNotFound      = "%s not found"
	ErrUnexpectedWorkflowFile        = "Unexpected workflow file: %s"
	ErrUnexpectedActionFound         = "Unexpected action: %s/%s@%s"

	// Additional test failures
	ErrWorkDirNotCreated             = "Work directory not created: %v"
	ErrWrongDirectoryPermissions     = "Wrong directory permissions: got %v, want %v"
	ErrWorkDirNotCleanedUp           = "Work directory not cleaned up properly"
	ErrFailedToReadGitConfig         = "Failed to read git config: %v"
	ErrGitConfigMissingValue         = "Git config missing expected value: %s"
	ErrExpectedDifferentPaths        = "Expected different repo paths for separate clones"
	ErrWorkflowFileNotFound          = "Workflow file not found in %s"
	ErrFailedToStatFile              = "Failed to stat file: %v"
	ErrWorkflowMissingContent        = "Workflow file missing expected content: %s"
	ErrFailedToChangeFilePermissions = "Failed to change file permissions: %v"
)
