package catalog

import (
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/manifest"
)

// Registry exposes package, configuration, and profile lookups backed by the declarative YAML manifest Catalog.
type Registry struct {
	cat *manifest.Catalog
}

// NewDefaultRegistry loads the embedded YAML manifests into a Registry.
func NewDefaultRegistry() *Registry {
	cat, err := manifest.LoadCatalog()
	if err != nil {
		// Embedded manifests are validated by unit tests at build time; panic indicates corrupted embed build.
		panic("failed to load embedded anx manifests: " + err.Error())
	}
	return &Registry{cat: cat}
}

// NewRegistryWithOverlays loads embedded YAML manifests overlaid with optional repository and user drop-in directories.
func NewRegistryWithOverlays(repoManifestDir, userDropInDir string) (*Registry, error) {
	cat, err := manifest.LoadCatalog(
		manifest.WithRepoManifestDir(repoManifestDir),
		manifest.WithUserDropInDir(userDropInDir),
	)
	if err != nil {
		return nil, err
	}
	return &Registry{cat: cat}, nil
}

// NewRegistryFromCatalog wraps an already loaded manifest.Catalog.
func NewRegistryFromCatalog(cat *manifest.Catalog) *Registry {
	if cat == nil {
		return NewDefaultRegistry()
	}
	return &Registry{cat: cat}
}

// Catalog returns the underlying declarative manifest.Catalog.
func (r *Registry) Catalog() *manifest.Catalog {
	return r.cat
}

// GetPackage finds a package by ID and converts it to a domain.ManagedResource.
func (r *Registry) GetPackage(id string) (domain.ManagedResource, bool) {
	spec, ok := r.cat.GetPackageSpec(id)
	if !ok {
		return domain.ManagedResource{}, false
	}
	return spec.ToManagedResource(), true
}

// GetPackageSpec returns the full declarative manifest.PackageSpec for id.
func (r *Registry) GetPackageSpec(id string) (manifest.PackageSpec, bool) {
	return r.cat.GetPackageSpec(id)
}

// ListPackages returns all registered packages sorted deterministically by ID.
func (r *Registry) ListPackages() []domain.ManagedResource {
	specs := r.cat.ListPackageSpecs()
	list := make([]domain.ManagedResource, 0, len(specs))
	for _, s := range specs {
		list = append(list, s.ToManagedResource())
	}
	return list
}

// GetConfiguration finds a configuration by ID.
func (r *Registry) GetConfiguration(id string) (domain.ConfigurationResource, bool) {
	return r.cat.GetConfiguration(id)
}

// AddConfiguration registers or updates a configuration in the registry.
func (r *Registry) AddConfiguration(cfg domain.ConfigurationResource) {
	r.cat.AddConfiguration(cfg)
}

// ListConfigurations returns all managed configurations sorted deterministically by ID.
func (r *Registry) ListConfigurations() []domain.ConfigurationResource {
	return r.cat.ListConfigurations()
}

// ListConfigurationsByGroup returns managed configurations filtered by group name (or default "all").
func (r *Registry) ListConfigurationsByGroup(group string) []domain.ConfigurationResource {
	return r.cat.ListConfigurationsByGroup(group)
}

// GetProfile finds a profile by name.
func (r *Registry) GetProfile(name string) (domain.Profile, bool) {
	return r.cat.GetProfile(name)
}

// ListProfiles returns all available profiles sorted deterministically by name.
func (r *Registry) ListProfiles() []domain.Profile {
	return r.cat.ListProfiles()
}

// ResolveProfilePackages returns the resolved package ID list including transitive inherited profiles.
func (r *Registry) ResolveProfilePackages(name string) ([]string, error) {
	return r.cat.ResolveProfilePackages(name)
}

// ResolveProfileConfigurations returns the resolved configuration list including inherited config_groups.
func (r *Registry) ResolveProfileConfigurations(name string) ([]domain.ConfigurationResource, error) {
	return r.cat.ResolveProfileConfigurations(name)
}

// ResolveInstallOrder sorts package IDs topologically according to depends_on edges.
func (r *Registry) ResolveInstallOrder(ids []string) ([]domain.ManagedResource, error) {
	specs, err := r.cat.ResolveInstallOrder(ids)
	if err != nil {
		return nil, err
	}
	res := make([]domain.ManagedResource, 0, len(specs))
	for _, s := range specs {
		res = append(res, s.ToManagedResource())
	}
	return res, nil
}

// ListArchives returns all retired or deprecated component archive records.
func (r *Registry) ListArchives() []manifest.ArchiveRecord {
	return r.cat.ListArchives()
}

// GetArchive returns an archive record by component ID.
func (r *Registry) GetArchive(id string) (manifest.ArchiveRecord, bool) {
	return r.cat.GetArchive(id)
}

// ListHacks returns all declarative hack profiles.
func (r *Registry) ListHacks() []domain.HackProfile {
	return r.cat.ListHacks()
}

// GetHack returns a hack profile by ID.
func (r *Registry) GetHack(id string) (domain.HackProfile, bool) {
	return r.cat.GetHack(id)
}
