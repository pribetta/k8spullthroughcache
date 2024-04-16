package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
	dockerclient "github.com/docker/docker/client"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

func createClientSet() *kubernetes.Clientset {
	k8sConfig := k8sconfig.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		panic(err.Error() + " line 32")
	}
	return clientSet
}

func HandleMain(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello Main"))
}

func AdmissionReviewRequest(w http.ResponseWriter, r *http.Request) admissionv1.AdmissionReview {
	universalDeserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err.Error() + " line 45")
	}
	var admissionReviewReq admissionv1.AdmissionReview

	_, _, err = universalDeserializer.Decode(body, nil, &admissionReviewReq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		panic(err.Error() + " line 52")
	}
	if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println("malformed admission review request: request is nil")
		panic("malformed admission review : request is nil")
	}

	return admissionReviewReq

}

func patchPod(img string) {

	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	dockerClient, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv)
	if err != nil {
		panic(err)
	}
	defer dockerClient.Close()

	svc := ecr.NewFromConfig(cfg)

	reg := "643930694730"

	s := strings.Split(img, ":")

	reponame := s[0]
	ecrreponame := strings.Split(s[0], "/")[1] + "/" + strings.Split(s[0], "/")[2]
	var reponames []string
	reponames = append(reponames, ecrreponame)
	tag := s[1]
	int_reg := strings.Split(reponame, "/")[0]
	if int_reg != "643930694730.dkr.ecr.us-east-1.amazonaws.com" {
		return
	}
	ext_reg := strings.Split(reponame, "/")[1]
	//int_reg := strings.Split(reponame, "/")[0]
	img_name := strings.Split(reponame, "/")[2]

	logger.Info(reponame + " " + ecrreponame + " " + ext_reg + " " + img_name)
	imgid := types.ImageIdentifier{
		ImageTag: &tag,
	}

	var imgarr []types.ImageIdentifier
	imgarr = append(imgarr, imgid)
	logger.Info("reponame is " + reponame)
	logger.Info("registry is " + reg)
	resp, err := svc.DescribeImages(ctx, &ecr.DescribeImagesInput{
		RepositoryName: &reponame,
		RegistryId:     &reg,
		ImageIds:       imgarr,
	})
	img_to_pull := "img"

	if err != nil {
		if ext_reg == "gcr-io" {
			img_to_pull = "gcr.io/google-containers/" + img_name + ":" + tag
			logger.Info(img_to_pull)
		} else if ext_reg == "registry-k8s-io" {
			img_to_pull = "registry.k8s.io/" + img_name + ":" + tag
			logger.Info(img_to_pull)
		} else if ext_reg == "docker-io" {
			img_to_pull = img_name + ":" + tag
			logger.Info(img_to_pull)
		} else if ext_reg == "ghcr-io" {
			img_to_pull = img_name + ":" + tag
			logger.Info(img_to_pull)
		} else {
			fmt.Println("pull through cache rules dont exist")
			panic(err.Error())
		}
		logger.Info("calling image pull")
		reader, err := dockerClient.ImagePull(ctx, img_to_pull, image.PullOptions{})
		io.Copy(os.Stdout, reader)
		if err != nil {
			fmt.Println("Unable to pull img")
			panic(err.Error())
		}
		logger.Info("image pull complete")
		err = dockerClient.ImageTag(ctx, img_to_pull, img)
		if err != nil {
			panic(err.Error())
		}
		logger.Info("image tag complete")
		_, err = svc.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{
			RepositoryNames: reponames,
			RegistryId:      &reg,
		})
		if err != nil {
			fmt.Println("repo doesnt exist")
			_, err := svc.CreateRepository(ctx, &ecr.CreateRepositoryInput{
				RepositoryName: &ecrreponame,
				RegistryId:     &reg,
			})
			if err != nil {
				panic(err.Error())
			}
		}
		var regarr []string
		regarr = append(regarr, reg)
		authresp, err := svc.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(*authresp.AuthorizationData[0].AuthorizationToken)
		decodedToken, err := base64.StdEncoding.DecodeString(*authresp.AuthorizationData[0].AuthorizationToken)
		fmt.Println(decodedToken)
		if err != nil {
			panic(err.Error())
		}

		authConfig := registry.AuthConfig{
			Username:      "AWS",
			Password:      strings.Split(string(decodedToken), ":")[1],
			ServerAddress: "https://643930694730.dkr.ecr.us-east-1.amazonaws.com",
		}

		authConfigBytes, err := json.Marshal(authConfig)
		if err != nil {
			panic(err.Error())
		}

		authConfigEncoded := base64.URLEncoding.EncodeToString(authConfigBytes)

		pushresp, err := dockerClient.ImagePush(ctx, img, image.PushOptions{
			RegistryAuth: authConfigEncoded,
		})

		if err != nil {
			panic(err.Error())
		}

		data, _ := io.ReadAll(pushresp)
		fmt.Println(string(data))

		fmt.Println("Image pushed")
		_, err = svc.DescribeImages(ctx, &ecr.DescribeImagesInput{
			RepositoryName: &ecrreponame,
			RegistryId:     &reg,
			ImageIds:       imgarr,
		})
		if err != nil {
			fmt.Println("img not found in ecr")
		} else {
			fmt.Println("img found in ECR")
		}

		_, err = dockerClient.ImageRemove(ctx, img, image.RemoveOptions{
			Force: true,
		})
		if err != nil {
			fmt.Println("couldnt remove image " + img)
		}

		_, err = dockerClient.ImageRemove(ctx, img_to_pull, image.RemoveOptions{
			Force: true,
		})
		if err != nil {
			fmt.Println("couldnt remove image " + img_to_pull)
		}

	} else {
		fmt.Println("Images:")
		for _, imgdet := range resp.ImageDetails {
			fmt.Println(*imgdet.ImageDigest)
		}
	}

}

func HandleMutate(w http.ResponseWriter, r *http.Request) {
	logger.Info("Handling mutate")
	clientset := createClientSet()
	pods, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error() + " line 81")
	}
	logger.Info("Number of pods running is " + strconv.Itoa(len(pods.Items)))

	admissionReviewReq := AdmissionReviewRequest(w, r)

	var pod corev1.Pod

	logger.Info("Calling unmarshal")
	err = json.Unmarshal(admissionReviewReq.Request.Object.Raw, &pod)

	if err != nil {
		panic(err.Error() + " line 92")
	}
	logger.Info("patching pod")

	image := pod.Spec.Containers[0].Image

	logger.Info("image " + image)

	patchPod(image)

	logger.Info("marshal pod")
	//patchBytes, err := json.Marshal(patch)

	if err != nil {
		panic(err.Error() + " line 100")
	}

	AdmissionReviewResponse := admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			UID:     admissionReviewReq.Request.UID,
			Allowed: true,
		},
	}
	//AdmissionReviewResponse.Response.Patch = patchBytes
	logger.Info("marshaling admissionreviewresp")

	bytes, err := json.Marshal(&AdmissionReviewResponse)

	if err != nil {
		panic(err.Error() + " line 114")
	}
	logger.Info("err!=nil write resp")

	logger.Info(bytes)

	w.Write(bytes)

}

func main() {
	var port int
	var certFile string
	var keyFile string
	flag.IntVar(&port, "port", 8443, "Webhook server port.")
	flag.StringVar(&certFile, "tlsCertFile", "/etc/webhook/certs/tls.crt", "File containing x509 cert for https")
	flag.StringVar(&keyFile, "tlsKeyFile", "/etc/webhook/certs/tls.key", "File containing the x509 private key to --tlsCertFile")
	flag.Parse()

	http.HandleFunc("/", HandleMain)
	http.HandleFunc("/mutate", HandleMutate)
	logger.Info("Starting server")
	log.Fatal(http.ListenAndServeTLS(":"+strconv.Itoa(port), certFile, keyFile, nil))
}
