<!-- BEGIN_TF_DOCS -->
## Requirements

No requirements.

## Providers

No providers.

## Modules

No modules.

## Resources

No resources.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_hotspare"></a> [hotspare](#input\_hotspare) | Hot-spare value. | `number` | `1` | no |
| <a name="input_protection_level"></a> [protection\_level](#input\_protection\_level) | Cluster data protection level. | `number` | `2` | no |
| <a name="input_stripe_width"></a> [stripe\_width](#input\_stripe\_width) | Stripe width = cluster\_size - protection\_level - 1 (by default). | `number` | `-1` | no |

## Outputs

No outputs.
<!-- END_TF_DOCS -->