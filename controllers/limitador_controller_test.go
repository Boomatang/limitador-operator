package controllers

import (
	"context"
	"time"

	"github.com/kuadrant/limitador-operator/pkg/limitador"

	limitadorv1alpha1 "github.com/kuadrant/limitador-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Limitador controller", func() {
	const (
		LimitadorNamespace = "default"
		LimitadorReplicas  = 2
		LimitadorImage     = "quay.io/3scale/limitador"
		LimitadorVersion   = "0.3.0"
		LimitadorHttpPort  = 8000
		LimitadorGttpPort  = 8001

		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	httpPortNumber := int32(LimitadorHttpPort)
	grpcPortNumber := int32(LimitadorGttpPort)

	replicas := LimitadorReplicas
	version := LimitadorVersion
	httpPort := &limitadorv1alpha1.TransportProtocol{Port: &httpPortNumber}
	grpcPort := &limitadorv1alpha1.TransportProtocol{Port: &grpcPortNumber}

	limits := []limitadorv1alpha1.RateLimit{
		{
			Conditions: []string{"req.method == GET"},
			MaxValue:   10,
			Namespace:  "test-namespace",
			Seconds:    60,
			Variables:  []string{"user_id"},
		},
		{
			Conditions: []string{"req.method == POST"},
			MaxValue:   5,
			Namespace:  "test-namespace",
			Seconds:    60,
			Variables:  []string{"user_id"},
		},
	}

	newLimitador := func() *limitadorv1alpha1.Limitador {
		// The name can't start with a number.
		name := "a" + string(uuid.NewUUID())

		return &limitadorv1alpha1.Limitador{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Limitador",
				APIVersion: "limitador.kuadrant.io/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: LimitadorNamespace,
			},
			Spec: limitadorv1alpha1.LimitadorSpec{
				Replicas: &replicas,
				Version:  &version,
				Listener: &limitadorv1alpha1.Listener{
					HTTP: httpPort,
					GRPC: grpcPort,
				},
				Limits: limits,
			},
		}
	}

	deletePropagationPolicy := client.PropagationPolicy(metav1.DeletePropagationForeground)

	Context("Creating a new empty Limitador object", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			limitadorObj.Spec = limitadorv1alpha1.LimitadorSpec{}
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("Should create a Limitador service with default ports", func() {
			createdLimitadorService := v1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&createdLimitadorService)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(len(createdLimitadorService.Spec.Ports)).Should(Equal(2))
			Expect(createdLimitadorService.Spec.Ports[0].Name).Should(Equal("http"))
			Expect(createdLimitadorService.Spec.Ports[0].Port).Should(Equal(limitadorv1alpha1.DefaultServiceHTTPPort))
			Expect(createdLimitadorService.Spec.Ports[1].Name).Should(Equal("grpc"))
			Expect(createdLimitadorService.Spec.Ports[1].Port).Should(Equal(limitadorv1alpha1.DefaultServiceGRPCPort))
		})
	})

	Context("Creating a new Limitador object", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("Should create a new deployment with the right number of replicas, version and config file", func() {
			createdLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&createdLimitadorDeployment)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(*createdLimitadorDeployment.Spec.Replicas).Should(
				Equal((int32)(LimitadorReplicas)),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Image).Should(
				Equal(LimitadorImage + ":" + LimitadorVersion),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].Env[1]).Should(
				Equal(v1.EnvVar{Name: "LIMITS_FILE", Value: "/home/limitador/etc/limitador-config.yaml", ValueFrom: nil}),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath).Should(
				Equal("/home/limitador/etc/"),
			)
			Expect(createdLimitadorDeployment.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).Should(
				Equal(limitador.LimitsCMNamePrefix + limitadorObj.Name),
			)
		})

		It("Should create a Limitador service", func() {
			createdLimitadorService := v1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&createdLimitadorService)
				return err == nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should build the correct Status", func() {
			createdLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() limitadorv1alpha1.LimitadorService {
				k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: limitadorObj.Namespace,
						Name:      limitadorObj.Name,
					},
					&createdLimitador)
				return createdLimitador.Status.Service
			}, timeout, interval).Should(Equal(limitadorv1alpha1.LimitadorService{
				Host: "limitador-" + limitadorObj.Name + ".default.svc.cluster.local",
				Ports: limitadorv1alpha1.Ports{
					GRPC: grpcPortNumber,
					HTTP: httpPortNumber,
				},
			}))

		})
		It("Should create a ConfigMap with the correct limits and hash", func() {
			createdConfigMap := v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.LimitsCMNamePrefix + limitadorObj.Name,
					},
					&createdConfigMap)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdConfigMap.Data[limitador.LimitadorCMHash]).Should(
				Equal("a00c9940ae6bb8de702633ce453e6a97"),
			)
			Expect(createdConfigMap.Data[limitador.LimitadorConfigFileName]).Should(
				Equal("- conditions:\n  - req.method == GET\n  max_value: 10\n  namespace: test-namespace\n  seconds: 60\n  variables:\n  - user_id\n- conditions:\n  - req.method == POST\n  max_value: 5\n  namespace: test-namespace\n  seconds: 60\n  variables:\n  - user_id\n"),
			)
		})
	})

	Context("Updating a limitador object", func() {
		var limitadorObj *limitadorv1alpha1.Limitador

		BeforeEach(func() {
			limitadorObj = newLimitador()
			err := k8sClient.Delete(context.TODO(), limitadorObj, deletePropagationPolicy)
			Expect(err == nil || errors.IsNotFound(err))

			Expect(k8sClient.Create(context.TODO(), limitadorObj)).Should(Succeed())
		})

		It("Should modify the limitador deployment", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			replicas = LimitadorReplicas + 1
			updatedLimitador.Spec.Replicas = &replicas
			version = "latest"
			updatedLimitador.Spec.Version = &version

			Expect(k8sClient.Update(context.TODO(), &updatedLimitador)).Should(Succeed())
			updatedLimitadorDeployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitadorDeployment)

				if err != nil {
					return false
				}

				correctReplicas := *updatedLimitadorDeployment.Spec.Replicas == LimitadorReplicas+1
				correctImage := updatedLimitadorDeployment.Spec.Template.Spec.Containers[0].Image == LimitadorImage+":latest"

				return correctReplicas && correctImage
			}, timeout, interval).Should(BeTrue())
		})
		It("Should modify the ConfigMap accordingly", func() {
			updatedLimitador := limitadorv1alpha1.Limitador{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitadorObj.Name,
					},
					&updatedLimitador)

				return err == nil
			}, timeout, interval).Should(BeTrue())

			limits := []limitadorv1alpha1.RateLimit{
				{
					Conditions: []string{"req.method == GET"},
					MaxValue:   100,
					Namespace:  "test-namespace",
					Seconds:    60,
					Variables:  []string{"user_id"},
				},
			}
			updatedLimitador.Spec.Limits = limits

			Expect(k8sClient.Update(context.TODO(), &updatedLimitador)).Should(Succeed())
			updatedLimitadorConfigMap := v1.ConfigMap{}
			Eventually(func() bool {
				err := k8sClient.Get(
					context.TODO(),
					types.NamespacedName{
						Namespace: LimitadorNamespace,
						Name:      limitador.LimitsCMNamePrefix + limitadorObj.Name,
					},
					&updatedLimitadorConfigMap)

				if err != nil {
					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())
			Expect(updatedLimitadorConfigMap.Data[limitador.LimitadorCMHash]).Should(Equal("69b3eab828208274d4200aedc6fd8b19"))
			Expect(updatedLimitadorConfigMap.Data[limitador.LimitadorConfigFileName]).Should(Equal("- conditions:\n  - req.method == GET\n  max_value: 100\n  namespace: test-namespace\n  seconds: 60\n  variables:\n  - user_id\n"))

		})
	})
})
