package e2e

import (
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-cd/errors"
	"github.com/argoproj/argo-cd/test/e2e/fixture"

	. "github.com/argoproj/argo-cd/pkg/apis/application/v1alpha1"
	. "github.com/argoproj/argo-cd/test/e2e/fixture/app"
)

func TestAutoSyncSelfHealDisabled(t *testing.T) {
	Given(t).
		Path(guestbookPath).
		When().
		// app should be auto-synced once created
		CreateFromFile(func(app *Application) {
			app.Spec.SyncPolicy = &SyncPolicy{Automated: &SyncPolicyAutomated{SelfHeal: false}}
		}).
		Then().
		Expect(SyncStatusIs(SyncStatusCodeSynced)).
		// app should be auto-synced if git change detected
		When().
		PatchFile("guestbook-ui-deployment.yaml", `[{"op": "replace", "path": "/spec/revisionHistoryLimit", "value": 1}]`).
		Refresh(RefreshTypeNormal).
		Then().
		Expect(SyncStatusIs(SyncStatusCodeSynced)).
		// app should not be auto-synced if k8s change detected
		When().
		And(func() {
			errors.FailOnErr(fixture.KubeClientset.AppsV1().Deployments(fixture.DeploymentNamespace()).Patch(
				"guestbook-ui", types.MergePatchType, []byte(`{"spec": {"revisionHistoryLimit": 0}}`)))
		}).
		Then().
		Expect(SyncStatusIs(SyncStatusCodeOutOfSync))
}

func TestAutoSyncSelfHealEnabled(t *testing.T) {
	Given(t).
		Path(guestbookPath).
		When().
		// app should be auto-synced once created
		CreateFromFile(func(app *Application) {
			app.Spec.SyncPolicy = &SyncPolicy{Automated: &SyncPolicyAutomated{SelfHeal: true}}
		}).
		Then().
		Expect(SyncStatusIs(SyncStatusCodeSynced)).
		When().
		// app should be auto-synced once k8s change detected
		And(func() {
			errors.FailOnErr(fixture.KubeClientset.AppsV1().Deployments(fixture.DeploymentNamespace()).Patch(
				"guestbook-ui", types.MergePatchType, []byte(`{"spec": {"revisionHistoryLimit": 0}}`)))
		}).
		Then().
		Expect(SyncStatusIs(SyncStatusCodeSynced)).
		When().
		// app should be attempted to auto-synced once and marked with error after failed attempt detected
		PatchFile("guestbook-ui-deployment.yaml", `[{"op": "replace", "path": "/spec/revisionHistoryLimit", "value": "badValue"}]`).
		Refresh(RefreshTypeNormal).
		Then().
		Expect(SyncStatusIs(SyncStatusCodeOutOfSync)).
		Expect(Condition(ApplicationConditionSyncError, "Failed sync attempt"))
}
