# Configuration Validation

- Define a new resource Config Schema which contains Helm Validation schema's 
  - since Helm is positioned as a primary templating approach for app developer, it makes sense to stick with the same approach
- App devs provide Config Schemas for the app configs and platform configs
   - App Config Schema is stored with the Helm Charts
   - App configs are validated by the CD pipeline with Helm while generating manifests
   - Platform Config Schema is, basically, a declaration of what platform configs are required by the workload
   - Platform Config Schema is stored in the folder with the Workload descriptor
   - Platform configs are delivered to the Kalypso Control plane cluster and used by the scheduler to validate platform configs
- Platform Devs provide Config Schemas for the platform configs in the control plane repository
  - They may declare for the further validation what config values, regardless of the workloads, 
      - should be provided for all environments - main branch
      - for a specific environment - env branch
  - Kalypso scheduler, while composing a platform config map for every assignment, validates config values against both platform and workload config schemas
    - In case of an error, it saves it to the assignment's status     
    - The errors are visible either as PR comments/checks or as repo issues or both

