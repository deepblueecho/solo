STATUS: DONE_WITH_CONCERNS

Files changed
- internal/server/service/artifact.go
- internal/server/service/artifact_test.go

Tests run with exact command and result
- `go test ./internal/server/service -run 'TestRenderArtifactHTML|TestArtifactFilenameForMode'` -> FAIL as expected before implementation; undefined `artifactRenderData`, `ArtifactTask`, `ArtifactMessage`, `renderArtifactHTML`, and `artifactFilename`.
- `gofmt -w internal/server/service/artifact.go internal/server/service/artifact_test.go`
- `go test ./internal/server/service -run 'TestRenderArtifactHTML|TestArtifactFilenameForMode'` -> PASS, `ok github.com/solo-ai/solo/internal/server/service 0.107s`.
- `gofmt -w internal/server/service/artifact.go internal/server/service/artifact_test.go && go test ./internal/server/service` -> PASS, `ok github.com/solo-ai/solo/internal/server/service 0.114s`.
- Fresh pre-commit verification: `gofmt -w internal/server/service/artifact.go internal/server/service/artifact_test.go && go test ./internal/server/service` -> PASS, `ok github.com/solo-ai/solo/internal/server/service 0.096s`.

Commits created
- cb5cdf4 Add task artifact renderer

Self-review notes
- Kept the code commit scoped to `internal/server/service/artifact.go` and `internal/server/service/artifact_test.go`.
- Implemented the required artifact service constructor, public methods, render helpers, message/thread loading, attachment metadata lookup, HTML file writing, artifact upsert, and source snapshot JSON.
- `Get` and `Latest` re-check task access through `TaskService` before returning artifact metadata.

Concerns, if any
- The brief's sample renderer used capitalized `Review before external sharing`, while the required test checks lowercase `review before external sharing`; the renderer uses the lowercase phrase so the exact required test passes.

Fix report after task review
- Changed artifact attachment rendering to metadata-only text. Artifact HTML no longer emits public `/api/v1/attachments/{id}` links.
- Added `TestRenderArtifactHTML_RendersAttachmentMetadataWithoutPublicLinks`.
- Tests run: `gofmt -w internal/server/service/artifact.go internal/server/service/artifact_test.go && go test ./internal/server/service`
- Result: PASS, `ok github.com/solo-ai/solo/internal/server/service 3.115s`.

Fix report after product decision update
- Restored artifact attachment URLs for existing in-product attachment routes.
- Image attachments now render inline with metadata; non-image attachments render as links with metadata.
- Tests run: `gofmt -w internal/server/service/artifact.go internal/server/service/artifact_test.go && go test ./internal/server/service`
- Result: PASS, `ok github.com/solo-ai/solo/internal/server/service 0.096s`.
