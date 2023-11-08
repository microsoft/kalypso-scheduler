# Configuration Validation

- App devs provide Config Schemas for the app configs and platform configs
   - App Config Schema is stored with the Helm Charts. Nothing new here. E.g. [values.schema.json](https://github.com/eedorenko/ConfiguratiX/blob/main/helmcharts/uc1-shared/values.schema.json)
   - App configs are validated by the CD pipeline with Helm while generating manifests
   - App developers have [workload.yaml](https://github.com/eedorenko/ConfiguratiX/blob/main/workloads/workload.yaml) which is basically a contract between app dev team and platform team. In this file app devs describe deployment targets of the application and declare with json schema what platform configs are required by the workload. They can define a schema for the entire workload and additionally for each deployment target.
   - Helm uses json schema for config validation and since Helm is positioned as the primary templating tool used by app devs, it makes sense for them to use the same approach (json schema) to describe platform configs     
- Platform Devs provide Config Schemas for the platform configs in the control plane repository
  - They may declare for the further validation what config values, regardless of the workloads (e.g. [configSchema.yaml](https://github.com/eedorenko/configuratix-kalypso-control-plane/blob/main/configs/configSchema.yaml)), 
      - should be provided for all environments - main branch 
      - for a specific environment - env branch
- Kalypso scheduler, while composing a platform config map for every assignment, validates config values against config schemas, defined by the app devs and by the schemas defined by the platform team if they satisfy the label matching with the cluster and deployment target. Same label matching mechanism which is used for config maps.
   - All validation errors Kalypso reports as Issues in Kalypso GitOps (e.g. [Issue](https://github.com/eedorenko/configuratix-kalypso-gitops/issues/272)) 
   - When the error is fixed, Kalypso automatically closes corresponding Issue 

