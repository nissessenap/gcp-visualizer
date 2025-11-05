---
date: 2025-11-05T20:02:52+01:00
researcher: Claude
git_commit: 72c578b4927b9d1a963e516668e33d471976984b
branch: main
repository: gcp-visualizer
topic: "GCP Resource Visualizer MVP Architecture Decisions"
tags: [research, architecture, gcp, visualization, pub-sub, service-accounts, graphviz, mvp]
status: complete
last_updated: 2025-11-05
last_updated_by: Claude
---

# Research: GCP Resource Visualizer MVP Architecture Decisions

**Date**: 2025-11-05T20:02:52+01:00
**Researcher**: Claude
**Git Commit**: 72c578b4927b9d1a963e516668e33d471976984b
**Branch**: main
**Repository**: gcp-visualizer

## Research Question

Research architectural decisions for building a GCP resource visualization CLI tool MVP that:
- Visualizes Service Account and Pub/Sub topic/subscription relationships
- Scales to 1000+ resources
- Avoids building a frontend (prefers existing visualization tools with APIs)
- Can later add Terraform management status visualization

## Summary

Based on comprehensive research, the recommended architecture combines:
1. **Data Collection**: Cloud Asset Inventory API with the Go SDK for fetching GCP resources and relationships
2. **Graph Generation**: Graphviz with go-graphviz library using the SFDP layout engine for scalable visualization
3. **Optional Enhancement**: Steampipe for SQL-based exploration and existing relationship queries

This approach provides a pure Go solution with no external dependencies that can handle 1000+ resources effectively.

## Detailed Findings

### Visualization Tools Analysis

#### Recommended: Graphviz with SFDP Engine

**Why Graphviz**:
- **Proven scalability**: SFDP engine handles 70,000+ nodes efficiently (tested with 1,054 nodes in seconds)
- **Pure Go integration**: go-graphviz provides embedded WASM implementation
- **No external dependencies**: Complete solution in Go
- **Multiple output formats**: SVG, PNG, PDF supported
- **30+ years of development**: Most mature graph visualization tool

**Implementation approach**:
```go
import "github.com/goccy/go-graphviz"

g := graphviz.New(ctx)
graph, _ := g.Graph()
// Build your GCP resource graph
graph.SetLayout(graphviz.SFDP) // Critical: Use sfdp for large graphs
g.RenderFilename(graph, graphviz.SVG, "output.svg")
```

**Key optimization settings for 1000+ nodes**:
- Use `sfdp` layout engine (NOT `dot` which struggles at 500+ nodes)
- Set `overlap=scale` for better node distribution
- Use `splines=line` for performance boost
- Adjust `K` parameter to control node spacing

#### Alternative for Small Visualizations: D2

If typical use case is under 500 nodes and aesthetics are priority:
- Modern, clean syntax
- Best-looking output with ELK layout engine
- Native Go library
- Active development
- **Explicit limitation**: Not tested on thousands of nodes

#### Not Recommended for This Use Case

**Mermaid**:
- Hard-coded 280 edge limit
- Performance degrades significantly with large graphs
- JavaScript/Node.js dependency

**PlantUML**:
- Requires Java runtime
- Slower performance on large graphs
- Less suitable for network diagrams

### GCP Data Collection Architecture

#### Primary Approach: Cloud Asset Inventory API

**Installation**:
```bash
go get cloud.google.com/go/asset/apiv1@latest
go get cloud.google.com/go/pubsub
```

**Key capabilities**:
- Organization-wide resource discovery with single API call
- RELATIONSHIP content type for tracking dependencies
- Cross-project resource access patterns
- 35-day history of changes
- Export to BigQuery for large-scale analysis

**Service Account Permission Discovery**:
```go
// Analyze IAM policies to find service account permissions
func analyzeSAPermissions(client *asset.Client, orgID, saEmail string) {
    req := &assetpb.AnalyzeIamPolicyRequest{
        AnalysisQuery: &assetpb.IamPolicyAnalysisQuery{
            Scope: fmt.Sprintf("organizations/%s", orgID),
            IdentitySelector: &assetpb.IamPolicyAnalysisQuery_IdentitySelector{
                Identity: fmt.Sprintf("serviceAccount:%s", saEmail),
            },
            Options: &assetpb.IamPolicyAnalysisQuery_Options{
                AnalyzeServiceAccountImpersonation: true,
                ExpandGroups: true,
                ExpandRoles: true,
            },
        },
    }

    resp, err := client.AnalyzeIamPolicy(ctx, req)
    // Process results showing what resources SA can access
}
```

**Pub/Sub Relationship Discovery**:
```go
// Discover topic-subscription relationships
func discoverPubSubRelationships(client *pubsub.Client) {
    // List all subscriptions and their topics
    subIter := client.Subscriptions(ctx)
    for {
        sub, err := subIter.Next()
        if err == iterator.Done {
            break
        }

        config, err := sub.Config(ctx)
        // config.Topic contains the topic reference
        fmt.Printf("Subscription: %s -> Topic: %s\n", sub.ID(), config.Topic.ID())
    }

    // List all topics and their subscriptions
    topicIter := client.Topics(ctx)
    for {
        topic, err := topicIter.Next()
        // topic.Subscriptions() lists all subscriptions
    }
}
```

**Cross-Project Discovery**:
```go
// Query resources across multiple projects from organization scope
req := &assetpb.SearchAllResourcesRequest{
    Scope: fmt.Sprintf("organizations/%s", orgID),
    AssetTypes: []string{
        "pubsub.googleapis.com/Topic",
        "pubsub.googleapis.com/Subscription",
    },
    PageSize: 500,
}
```

#### API Quotas and Rate Limits

**Critical quotas for large-scale discovery**:

| Operation | Project Limit/min | Org Limit/min |
|-----------|------------------|---------------|
| SearchAllResources | 400 | 1,500 |
| ListAssets | 100 | 800 |
| AnalyzeIamPolicy | 100 | - |

**Rate limiting strategy**:
- Implement exponential backoff for ResourceExhausted errors
- Use BigQuery exports for batch processing (doesn't count against quotas)
- Partition queries by project/folder
- Request quota increases for Premium/Enterprise tier

### Existing Tools Evaluation

#### Most Relevant: Steampipe + GCP Plugin

**Advantages**:
- 121+ GCP resource types including service accounts and Pub/Sub
- SQL-based querying without database requirement
- GCP Insights Mod provides pre-built relationship graphs
- Can be integrated into CLI tools
- GraphQL endpoint for programmatic access

**Integration path**:
1. Use Steampipe to query GCP data via SQL
2. Build Go CLI that queries Steampipe
3. Generate visualizations from query results

#### Alternative: CloudGraph

**Advantages**:
- GraphQL API designed for resource relationships
- Stores in Dgraph (graph database)
- Full relationship mapping between resources
- Docker-based deployment

**Limitations**:
- Requires Docker and Dgraph
- More complex setup than direct API approach

#### Security-Focused: GCP Scanner

**Advantages**:
- Assesses service account credential access levels
- Built-in visualizer (web-based)
- Supports Pub/Sub, Service Accounts, and other GCP resources
- Python-based with JSON output

**Limitations**:
- Security-focused rather than general visualization
- Linux-only
- Would require Python integration

## Architecture Recommendations

### MVP Architecture (Recommended)

```
┌─────────────────┐
│   Go CLI Tool   │
└────────┬────────┘
         │
    ┌────▼────┐
    │ Collect │ (Cloud Asset Inventory API + Pub/Sub API)
    └────┬────┘
         │
    ┌────▼────┐
    │  Build  │ (Internal graph structure)
    └────┬────┘
         │
    ┌────▼────┐
    │ Render  │ (go-graphviz with SFDP engine)
    └────┬────┘
         │
    ┌────▼────┐
    │ Output  │ (SVG/PNG/PDF files)
    └─────────┘
```

**Implementation Steps**:

1. **Data Collection Module**:
   - Use Cloud Asset Inventory API for service account IAM bindings
   - Use Pub/Sub API for topic/subscription relationships
   - Handle cross-project resources via organization-level queries

2. **Graph Building Module**:
   - Create internal graph representation
   - Track nodes (service accounts, topics, subscriptions)
   - Track edges (publishes-to, subscribes-to, has-permission-on)

3. **Visualization Module**:
   - Use go-graphviz with SFDP layout for 1000+ nodes
   - Implement clustering by project or service
   - Support multiple output formats

4. **Configuration**:
   - YAML/JSON config for filtering resources
   - Layout preferences
   - Output format selection

### Hybrid Approach (Future Enhancement)

For enhanced capabilities, consider adding:

1. **Steampipe Integration**:
   - Use for SQL-based exploration
   - Pre-built relationship queries
   - Interactive dashboards

2. **BigQuery Export**:
   - For organizations with 10,000+ resources
   - Complex relationship analysis via SQL
   - Historical tracking

3. **D2 for Small Graphs**:
   - Auto-detect graph size
   - Use D2 for <500 nodes (better aesthetics)
   - Use Graphviz for 500+ nodes (better performance)

## Implementation Considerations

### Required IAM Permissions

Minimum permissions for the tool's service account:
- `roles/cloudasset.viewer` - Read asset metadata
- `roles/iam.securityReviewer` - Analyze IAM policies
- `roles/pubsub.viewer` - List topics and subscriptions
- Organization-level access for cross-project discovery

### Performance Optimizations

1. **Parallel Processing**:
   - Query multiple projects concurrently
   - Batch API requests where possible

2. **Caching**:
   - Cache discovered resources locally
   - Implement TTL for refresh

3. **Filtering**:
   - Allow resource type filtering
   - Project/folder scoping
   - Label-based filtering

4. **Progressive Rendering**:
   - Start with high-level view
   - Allow drill-down into specific services

### Terraform Integration (Phase 2)

For detecting Terraform-managed resources:
- Parse Terraform state files (`.tfstate`)
- Use resource IDs to match with discovered resources
- Add visual indicators (color, shape, label) for managed resources

## Cost Considerations

1. **Cloud Asset Inventory API**:
   - First 1 million API calls free per month
   - $0.01 per 1,000 API calls after

2. **Pub/Sub API**:
   - Admin operations are free
   - No charges for listing/describing

3. **BigQuery (if used)**:
   - $5 per TB for queries
   - Storage costs for exported data

## Open Questions

1. **Resource Grouping Strategy**: How should resources be grouped in large visualizations? By project, service, or custom tags?

2. **Interactive Features**: Is a static image sufficient, or would interactive HTML with zoom/pan be valuable?

3. **Update Frequency**: How often should the visualization be regenerated? Real-time, daily, or on-demand?

4. **Access Pattern Details**: Should the tool show specific IAM roles/permissions or just connection existence?

5. **Terraform State Location**: Where are Terraform state files stored? GCS, local, or Terraform Cloud?

## Follow-up Research Topics

1. **Graph Layout Algorithms**: Research optimal layouts for service dependency graphs
2. **Incremental Updates**: Methods for updating visualizations without full regeneration
3. **Multi-Cloud Extension**: Feasibility of extending to AWS/Azure
4. **Cost Optimization**: Strategies for minimizing API calls in large organizations

## Related Research

- Graphviz documentation: https://graphviz.org/documentation/
- Cloud Asset Inventory best practices: https://cloud.google.com/asset-inventory/docs/best-practices
- go-graphviz examples: https://github.com/goccy/go-graphviz/tree/main/examples
- Steampipe GCP plugin: https://hub.steampipe.io/plugins/turbot/gcp

## Conclusion

The recommended MVP architecture using Cloud Asset Inventory API + go-graphviz provides a solid foundation that:
- Handles 1000+ resources effectively
- Requires no external dependencies
- Scales from project to organization level
- Can be extended with additional features

The key technical decision is using Graphviz's SFDP layout engine instead of the default DOT engine, which is critical for performance at scale.