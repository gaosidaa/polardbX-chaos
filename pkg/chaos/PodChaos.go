package chaos

import "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"

type ChaosTemplateBase struct {
	Name           string
	Mode           v1alpha1.SelectorMode
	LabelSelectors map[string]string
	Namespaces     []string
	Deadline       string
	Action         string
}

func NewChaosTemplateBase(name, deadline string, namespaces []string) *ChaosTemplateBase {
	return &ChaosTemplateBase{Name: name,
		Deadline:   deadline,
		Namespaces: namespaces,
		Mode:       v1alpha1.OneMode,
		Action:     string(v1alpha1.PodKillAction),
	}
}

func (this ChaosTemplateBase) PodChaosInit() v1alpha1.Template {
	return v1alpha1.Template{
		Name:     this.Name,
		Type:     v1alpha1.KindPodChaos,
		Deadline: &this.Deadline,
		EmbedChaos: &v1alpha1.EmbedChaos{
			PodChaos: &v1alpha1.PodChaosSpec{
				ContainerSelector: v1alpha1.ContainerSelector{
					PodSelector: v1alpha1.PodSelector{
						Mode: this.Mode,
						Selector: v1alpha1.PodSelectorSpec{
							GenericSelectorSpec: v1alpha1.GenericSelectorSpec{
								LabelSelectors: this.LabelSelectors,
								Namespaces:     this.Namespaces,
							},
						},
					},
				},
				Action:      v1alpha1.PodChaosAction(this.Action),
				GracePeriod: 0,
			},
		},
	}
}
func (this ChaosTemplateBase) IoChaosInit(delay, volumePath string, percent int) v1alpha1.Template {
	return v1alpha1.Template{
		Name:     this.Name,
		Type:     v1alpha1.KindIOChaos,
		Deadline: &this.Deadline,
		EmbedChaos: &v1alpha1.EmbedChaos{
			IOChaos: &v1alpha1.IOChaosSpec{
				ContainerSelector: v1alpha1.ContainerSelector{
					PodSelector: v1alpha1.PodSelector{

						Mode: this.Mode,
						Selector: v1alpha1.PodSelectorSpec{
							GenericSelectorSpec: v1alpha1.GenericSelectorSpec{
								LabelSelectors: this.LabelSelectors,
								Namespaces:     this.Namespaces,
							},
						},
					},
				},
				Action:     v1alpha1.IOChaosType(this.Action),
				Delay:      delay,
				Percent:    percent,
				VolumePath: volumePath,
			},
		},
	}
}
func (this ChaosTemplateBase) NetworkChaosInit(delay *v1alpha1.DelaySpec,) v1alpha1.Template {
	return v1alpha1.Template{
		Name:     this.Name,
		Type:     v1alpha1.KindNetworkChaos,
		Deadline: &this.Deadline,
		EmbedChaos: &v1alpha1.EmbedChaos{
			
			NetworkChaos: &v1alpha1.NetworkChaosSpec{
				TcParameter: v1alpha1.TcParameter{
					Delay: delay,
				},
				Target: &v1alpha1.PodSelector{
					Selector: v1alpha1.PodSelectorSpec{
						GenericSelectorSpec: v1alpha1.GenericSelectorSpec{
							LabelSelectors: this.LabelSelectors,
							Namespaces:     this.Namespaces,
						},
					},
					Mode: this.Mode,
				},
				Direction:  v1alpha1.Both,
				Action: v1alpha1.DelayAction,
				PodSelector: v1alpha1.PodSelector{
					Mode: this.Mode,
					Selector: v1alpha1.PodSelectorSpec{
						GenericSelectorSpec: v1alpha1.GenericSelectorSpec{
							LabelSelectors: this.LabelSelectors,
							Namespaces:     this.Namespaces,
						},
						
					},
				},
			},
		},
	}
}


func (this ChaosTemplateBase) SetLabels(label map[string]string) ChaosTemplateBase {
	this.LabelSelectors = label
	return this
}
func (this ChaosTemplateBase) SetAction(Action string) ChaosTemplateBase {
	this.Action = Action
	return this
}
func (this ChaosTemplateBase) SetMode(mode v1alpha1.SelectorMode) ChaosTemplateBase {
	this.Mode = mode
	return this
}
