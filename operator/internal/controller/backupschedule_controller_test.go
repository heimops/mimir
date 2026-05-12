package controller_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mimirio "mimir.io/operator/api/v1alpha1"
)

// findCondition returns the condition with the given type, or nil if not found.
func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

var _ = Describe("BackupSchedule controller", func() {
	const (
		timeout  = "10s"
		interval = "100ms"
	)

	// helper: waits until a CronJob with the given name exists in the namespace
	getCronJob := func(ctx context.Context, namespace, name string) func(g Gomega) *batchv1.CronJob {
		return func(g Gomega) *batchv1.CronJob {
			cj := &batchv1.CronJob{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, cj)).To(Succeed())
			return cj
		}
	}

	// helper: builds a minimal BackupSchedule
	newBS := func(name, namespace string, spec mimirio.BackupScheduleSpec) *mimirio.BackupSchedule {
		return &mimirio.BackupSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec:       spec,
		}
	}

	Context("when a BackupSchedule is created", func() {
		It("creates a CronJob with the correct schedule and env vars", func() {
			ctx := context.Background()
			bs := newBS("bs-basic", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 2 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			var cj *batchv1.CronJob
			Eventually(getCronJob(ctx, "default", "mimir-bs-basic"), timeout, interval).
				Should(And(Not(BeNil()), WithTransform(func(c *batchv1.CronJob) string {
					return c.Spec.Schedule
				}, Equal("0 2 * * *"))))

			cj = getCronJob(ctx, "default", "mimir-bs-basic")(Default)
			container := cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0]

			Expect(container.Image).To(Equal("mimir:test"))
			Expect(container.Args).To(Equal([]string{"backup"}))
			Expect(container.EnvFrom).To(BeEmpty())

			envMap := envVarMap(container.Env)
			Expect(envMap).To(HaveKeyWithValue("MIMIR_NAMESPACES", "production"))
			Expect(envMap).To(HaveKeyWithValue("MIMIR_STORAGE_BACKEND", "s3"))
			Expect(envMap).NotTo(HaveKey("MIMIR_PVCS"))
		})

		It("sets MIMIR_PVCS when specific PVCs are listed", func() {
			ctx := context.Background()
			bs := newBS("bs-pvcs", "default", mimirio.BackupScheduleSpec{
				Namespace:      "staging",
				Schedule:       "0 3 * * *",
				StorageBackend: "azure",
				PVCs:           []string{"db-data", "uploads"},
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(func(g Gomega) {
				cj := getCronJob(ctx, "default", "mimir-bs-pvcs")(g)
				env := envVarMap(cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Env)
				g.Expect(env).To(HaveKeyWithValue("MIMIR_PVCS", "db-data,uploads"))
			}, timeout, interval).Should(Succeed())
		})

		It("adds envFrom when storageSecret is set", func() {
			ctx := context.Background()
			bs := newBS("bs-secret", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 4 * * *",
				StorageBackend: "gcs",
				StorageSecret:  "mimir-gcs-creds",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(func(g Gomega) {
				cj := getCronJob(ctx, "default", "mimir-bs-secret")(g)
				envFrom := cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].EnvFrom
				g.Expect(envFrom).To(HaveLen(1))
				g.Expect(envFrom[0].SecretRef.Name).To(Equal("mimir-gcs-creds"))
			}, timeout, interval).Should(Succeed())
		})

		It("uses the image override from the spec", func() {
			ctx := context.Background()
			bs := newBS("bs-image", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 5 * * *",
				StorageBackend: "s3",
				Image:          "myregistry.io/mimir:v1.2.3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(func(g Gomega) {
				cj := getCronJob(ctx, "default", "mimir-bs-image")(g)
				g.Expect(cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image).
					To(Equal("myregistry.io/mimir:v1.2.3"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("when a BackupSchedule is updated", func() {
		It("propagates the new schedule to the CronJob", func() {
			ctx := context.Background()
			bs := newBS("bs-update", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 1 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(getCronJob(ctx, "default", "mimir-bs-update"), timeout, interval).
				ShouldNot(BeNil())

			// Update the schedule
			patch := client.MergeFrom(bs.DeepCopy())
			bs.Spec.Schedule = "0 6 * * *"
			Expect(k8sClient.Patch(ctx, bs, patch)).To(Succeed())

			Eventually(func(g Gomega) {
				cj := getCronJob(ctx, "default", "mimir-bs-update")(g)
				g.Expect(cj.Spec.Schedule).To(Equal("0 6 * * *"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("when a BackupSchedule is deleted", func() {
		It("garbage-collects the CronJob via owner reference", func() {
			ctx := context.Background()
			bs := newBS("bs-delete", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 7 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())

			Eventually(getCronJob(ctx, "default", "mimir-bs-delete"), timeout, interval).
				ShouldNot(BeNil())

			Expect(k8sClient.Delete(ctx, bs)).To(Succeed())

			Eventually(func() error {
				cj := &batchv1.CronJob{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name: "mimir-bs-delete", Namespace: "default",
				}, cj)
			}, timeout, interval).Should(MatchError(ContainSubstring("not found")))
		})
	})

	Context("CronJob name truncation", func() {
		It("truncates names longer than 52 chars", func() {
			ctx := context.Background()
			longName := fmt.Sprintf("a%s", repeatStr("x", 60))
			bs := newBS(longName, "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 8 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			expectedName := fmt.Sprintf("mimir-%s", longName)[:52]
			Eventually(getCronJob(ctx, "default", expectedName), timeout, interval).
				ShouldNot(BeNil())
		})
	})

	Context("ConcurrencyPolicy", func() {
		It("sets Forbid to prevent overlapping runs", func() {
			ctx := context.Background()
			bs := newBS("bs-concurrency", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 9 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(func(g Gomega) {
				cj := getCronJob(ctx, "default", "mimir-bs-concurrency")(g)
				g.Expect(cj.Spec.ConcurrencyPolicy).To(Equal(batchv1.ForbidConcurrent))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("status", func() {
		It("reflects the CronJob name in BackupSchedule status", func() {
			ctx := context.Background()
			bs := newBS("bs-status", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 10 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(func(g Gomega) {
				updated := &mimirio.BackupSchedule{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: "bs-status", Namespace: "default",
				}, updated)).To(Succeed())
				g.Expect(updated.Status.CronJob).To(Equal("mimir-bs-status"))
			}, timeout, interval).Should(Succeed())
		})

		It("sets Ready=True after a successful sync", func() {
			ctx := context.Background()
			bs := newBS("bs-ready", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 13 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(func(g Gomega) {
				updated := &mimirio.BackupSchedule{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: "bs-ready", Namespace: "default",
				}, updated)).To(Succeed())

				cond := findCondition(updated.Status.Conditions, mimirio.ConditionReady)
				g.Expect(cond).NotTo(BeNil())
				g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
				g.Expect(cond.Reason).To(Equal(mimirio.ReasonCronJobSynced))
				g.Expect(cond.ObservedGeneration).To(Equal(updated.Generation))
			}, timeout, interval).Should(Succeed())
		})

		It("transitions Ready back to True after a spec update", func() {
			ctx := context.Background()
			bs := newBS("bs-retransition", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 14 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			// Wait for initial Ready=True
			Eventually(func(g Gomega) {
				updated := &mimirio.BackupSchedule{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: "bs-retransition", Namespace: "default",
				}, updated)).To(Succeed())
				cond := findCondition(updated.Status.Conditions, mimirio.ConditionReady)
				g.Expect(cond).NotTo(BeNil())
				g.Expect(string(cond.Status)).To(Equal("True"))
			}, timeout, interval).Should(Succeed())

			// Update the schedule
			patch := client.MergeFrom(bs.DeepCopy())
			bs.Spec.Schedule = "0 15 * * *"
			Expect(k8sClient.Patch(ctx, bs, patch)).To(Succeed())

			// Condition must remain True (generation advances)
			Eventually(func(g Gomega) {
				updated := &mimirio.BackupSchedule{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: "bs-retransition", Namespace: "default",
				}, updated)).To(Succeed())
				cond := findCondition(updated.Status.Conditions, mimirio.ConditionReady)
				g.Expect(cond).NotTo(BeNil())
				g.Expect(string(cond.Status)).To(Equal("True"))
				g.Expect(cond.ObservedGeneration).To(Equal(updated.Generation))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("ServiceAccount", func() {
		It("assigns the mimir service account to the Job pod", func() {
			ctx := context.Background()
			bs := newBS("bs-sa", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 11 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(func(g Gomega) {
				cj := getCronJob(ctx, "default", "mimir-bs-sa")(g)
				g.Expect(cj.Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName).
					To(Equal("mimir"))
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("RestartPolicy", func() {
		It("sets OnFailure so failed pods are retried without creating a new Job", func() {
			ctx := context.Background()
			bs := newBS("bs-restart", "default", mimirio.BackupScheduleSpec{
				Namespace:      "production",
				Schedule:       "0 12 * * *",
				StorageBackend: "s3",
			})
			Expect(k8sClient.Create(ctx, bs)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(ctx, bs) })

			Eventually(func(g Gomega) {
				cj := getCronJob(ctx, "default", "mimir-bs-restart")(g)
				g.Expect(cj.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy).
					To(Equal(corev1.RestartPolicyOnFailure))
			}, timeout, interval).Should(Succeed())
		})
	})
})

// envVarMap converts a slice of EnvVar to a map for easier assertions.
func envVarMap(vars []corev1.EnvVar) map[string]string {
	m := make(map[string]string, len(vars))
	for _, v := range vars {
		m[v.Name] = v.Value
	}
	return m
}

func repeatStr(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
