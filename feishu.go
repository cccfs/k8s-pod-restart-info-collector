package main

import (
	"context"
	"fmt"
	lark "github.com/larksuite/oapi-sdk-go/v2"
	"github.com/lithammer/dedent"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type Feishu struct {
	WebhookUrl   string
	DefaultRobot string // Feishu Robot id
	ClusterName  string // Kubernete cluster name (will show in feishu message)
	MuteSeconds  int    // The time to mute duplicate alerts
	// History stores sent alerts, key: Namespace/podName, value: sentTime
	History         map[string]time.Time
	ClusterID       string
	DrcloudPlatform string
}

type FeishuMessage struct {
	Title   string
	Text    string
	Address string
}

func NewFeishu() Feishu {
	var feishuWebhookUrl, feishuRobot, clusterName, clusterID, drcloudPlatform string

	if feishuWebhookUrl = os.Getenv("FEISHU_WEBHOOK_URL"); feishuWebhookUrl == "" {
		feishuWebhookUrl = "https://open.feishu.cn/open-apis/bot/v2/hook"
		klog.Warningf("Environment variable FEISHU_WEBHOOK_URL is not set, default: %s\n", feishuWebhookUrl)
	}

	if feishuRobot = os.Getenv("FEISHU_ROBOT"); feishuRobot == "" {
		feishuRobot = "8366b2d1-ffc8-4081-8f2a-2f26c1bd114b"
		klog.Warningf("Environment variable FEISHU_ROBOT is not set, default: %s\n", feishuRobot)
	}

	if clusterName = os.Getenv("CLUSTER_NAME"); clusterName == "" {
		clusterName = "none"
		klog.Warningf("Environment variable CLUSTER_NAME is not set, default: %s\n", clusterName)
	}

	if clusterID = os.Getenv("CLUSTER_ID"); clusterID == "" {
		clusterID = "none"
		klog.Warningf("Environment variable CLUSTER_ID is not set, default: %s\n", clusterID)
	}

	if drcloudPlatform = os.Getenv("DRCLOUD_PLATFORM"); drcloudPlatform == "" {
		drcloudPlatform = "https://dev-drcloud.srv.deeproute.cn"
		klog.Warningf("Environment variable DRCLOUD_PLATFORM is not set, default: %s\n", drcloudPlatform)
	}

	muteSeconds, err := strconv.Atoi(os.Getenv("MUTE_SECONDS"))
	if err != nil {
		muteSeconds = 600
		klog.Warningf("Environment variable MUTE_SECONDS is not set, default: %d\n", muteSeconds)
	}

	klog.Infof("feishu Info: clustername: %s, muteseconds: %d\n", clusterName, muteSeconds)

	return Feishu{
		WebhookUrl:      feishuWebhookUrl,
		DefaultRobot:    feishuRobot,
		ClusterName:     clusterName,
		ClusterID:       clusterID,
		DrcloudPlatform: drcloudPlatform,
		MuteSeconds:     muteSeconds,
		History:         make(map[string]time.Time),
	}
}

func (f Feishu) sendToRobot(msg FeishuMessage, feishuRobot, serviceCreator string) error {
	robot := f.DefaultRobot
	if feishuRobot != "" {
		robot = feishuRobot
	}
	tmpl := `{
	"schema": "2.0",
	"config": {
		"update_multi": true,
		"style": {
			"text_size": {
				"normal_v2": {
					"default": "normal",
					"pc": "normal",
					"mobile": "heading"
				}
			},
			"color": {
				"color_avx80lmgh1m": {
					"light_mode": "rgba(248, 250, 255, 1)",
					"dark_mode": "rgba(10, 19, 41, 1)"
				},
				"color_3u5dhsmb3ig": {
					"light_mode": "rgba(248, 250, 255, 1)",
					"dark_mode": "rgba(10, 19, 41, 1)"
				}
			}
		}
	},
	"body": {
		"direction": "vertical",
		"horizontal_spacing": "8px",
		"vertical_spacing": "8px",
		"horizontal_align": "left",
		"vertical_align": "top",
		"padding": "0px 12px 12px 12px",
		"elements": [
			{
				"tag": "interactive_container",
				"width": "fill",
				"height": "auto",
				"corner_radius": "12px",
				"elements": [
					{
						"tag": "markdown",
						"content": "{{ .TITLE }}",
						"text_align": "left",
						"text_size": "normal_v2",
						"margin": "0px 0px 0px 20px"
					},
					{
						"tag": "hr"
					},
					{
						"tag": "column_set",
						"horizontal_spacing": "8px",
						"horizontal_align": "left",
						"columns": [
							{
								"tag": "column",
								"width": "weighted",
								"elements": [
									{
										"tag": "markdown",
										"content": "{{ .TEXT }}",
										"text_align": "left",
										"text_size": "normal_v2",
										"margin": "0px 0px 0px 0px"
									}
								],
								"padding": "0px 0px 12px 0px",
								"direction": "vertical",
								"horizontal_spacing": "8px",
								"vertical_spacing": "8px",
								"horizontal_align": "left",
								"vertical_align": "top",
								"margin": "0px 0px 0px 20px",
								"weight": 3
							}
						],
						"margin": "0px 0px 0px 0px"
					}
				],
				"has_border": true,
				"border_color": "blue-100",
				"background_style": "color_3u5dhsmb3ig",
				"padding": "12px 0px 0px 0px",
				"direction": "vertical",
				"horizontal_spacing": "8px",
				"vertical_spacing": "12px",
				"horizontal_align": "left",
				"vertical_align": "top",
				"margin": "0px 0px 0px 0px"
			},
			{
				"tag": "column_set",
				"horizontal_spacing": "8px",
				"horizontal_align": "left",
				"columns": [
					{
						"tag": "column",
						"width": "auto",
						"elements": [
							{
								"tag": "button",
								"text": {
									"tag": "plain_text",
									"content": "查看Pod监控"
								},
								"type": "primary_text",
								"width": "default",
								"size": "medium",
								"icon": {
									"tag": "standard_icon",
									"token": "no_outlined"
								},
								"behaviors": [
									{
										"type": "open_url",
										"default_url": "{{ .ADDRESS }}",
										"pc_url": "",
										"ios_url": "",
										"android_url": ""
									}
								]
							}
						],
						"vertical_spacing": "8px",
						"horizontal_align": "left",
						"vertical_align": "top"
					}
				]
			}
		]
	},
	"header": {
		"title": {
			"tag": "plain_text",
			"content": "Pod Crash Notify"
		},
		"subtitle": {
			"tag": "plain_text",
			"content": ""
		},
		"template": "default",
		"icon": {
			"tag": "custom_icon",
			"img_key": "img_v3_02ns_01458e14-8e2e-47bb-9f81-19a6d3b0a3ag"
		},
		"padding": "12px 12px 12px 12px"
	}
}`
	tmplMsg, _ := template.New("msg").Parse(
		dedent.Dedent(tmpl))
	card, err := Render(tmplMsg, Data{
		"TITLE":   msg.Title,
		"TEXT":    msg.Text,
		"ADDRESS": msg.Address,
	})
	if err != nil {
		return err
	}

	customerBot := lark.NewCustomerBot(fmt.Sprintf("%s/%s", f.WebhookUrl, robot), "")
	resp, err := customerBot.SendMessage(context.TODO(), "interactive", card)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return err
	}
	klog.Infof("Sent: [%s] to Feishu.\n\n", strings.Replace(msg.Title, "\n", " ", -1))
	return nil
}

type Data map[string]interface{}

func Render(tmpl *template.Template, variables map[string]interface{}) (string, error) {
	var buf strings.Builder

	if err := tmpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("failed to render template")
	}
	return buf.String(), nil
}
