package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/facette/facette/pkg/library"
	"github.com/facette/facette/pkg/utils"
	"github.com/facette/facette/thirdparty/github.com/fatih/set"
)

func (server *Server) serveCatalog(writer http.ResponseWriter, request *http.Request) {
	setHTTPCacheHeaders(writer)

	if request.URL.Path == urlCatalogPath {
		server.serveFullCatalog(writer, request)
	} else if strings.HasPrefix(request.URL.Path, urlCatalogPath+"origins/") {
		server.serveOrigin(writer, request)
	} else if strings.HasPrefix(request.URL.Path, urlCatalogPath+"sources/") {
		server.serveSource(writer, request)
	} else if strings.HasPrefix(request.URL.Path, urlCatalogPath+"metrics/") {
		server.serveMetric(writer, request)
	} else {
		server.serveResponse(writer, nil, http.StatusNotFound)
	}
}

func (server *Server) serveFullCatalog(writer http.ResponseWriter, request *http.Request) {
	catalog := make(map[string]map[string][]string)

	for originName, origin := range server.Catalog.Origins {
		catalog[originName] = make(map[string][]string)

		for sourceName, sources := range origin.Sources {
			catalog[originName][sourceName] = make([]string, 0)

			for metricName := range sources.Metrics {
				catalog[originName][sourceName] = append(catalog[originName][sourceName], metricName)
			}

			sort.Strings(catalog[originName][sourceName])
		}
	}

	server.serveResponse(writer, catalog, http.StatusOK)
}

func (server *Server) serveOrigin(writer http.ResponseWriter, request *http.Request) {
	originName := strings.TrimPrefix(request.URL.Path, urlCatalogPath+"origins/")

	if originName == "" {
		server.serveOriginList(writer, request)
		return
	}

	if response, status := server.parseShowRequest(writer, request); status != http.StatusOK {
		server.serveResponse(writer, response, status)
		return
	} else if _, ok := server.Catalog.Origins[originName]; !ok {
		server.serveResponse(writer, serverResponse{mesgResourceNotFound}, http.StatusNotFound)
		return
	}

	response := OriginResponse{
		Name:      originName,
		Connector: server.Config.Origins[originName].Connector["type"].(string),
		Updated:   server.Catalog.Updated.Format(time.RFC3339),
	}

	server.serveResponse(writer, response, http.StatusOK)
}

func (server *Server) serveOriginList(writer http.ResponseWriter, request *http.Request) {
	var offset, limit int

	if response, status := server.parseListRequest(writer, request, &offset, &limit); status != http.StatusOK {
		server.serveResponse(writer, response, status)
		return
	}

	originSet := set.New(set.ThreadSafe)

	for _, origin := range server.Catalog.Origins {
		if request.FormValue("filter") != "" && !utils.FilterMatch(request.FormValue("filter"), origin.Name) {
			continue
		}

		originSet.Add(origin.Name)
	}

	response := &listResponse{
		list:   StringListResponse(set.StringSlice(originSet)),
		offset: offset,
		limit:  limit,
	}

	server.applyResponseLimit(writer, request, response)

	server.serveResponse(writer, response.list, http.StatusOK)
}

func (server *Server) serveSource(writer http.ResponseWriter, request *http.Request) {
	sourceName := strings.TrimPrefix(request.URL.Path, urlCatalogPath+"sources/")

	if sourceName == "" {
		server.serveSourceList(writer, request)
		return
	} else if response, status := server.parseShowRequest(writer, request); status != http.StatusOK {
		server.serveResponse(writer, response, status)
		return
	}

	originSet := set.New(set.ThreadSafe)

	for _, origin := range server.Catalog.Origins {
		if _, ok := origin.Sources[sourceName]; ok {
			originSet.Add(origin.Name)
		}
	}

	if originSet.Size() == 0 {
		server.serveResponse(writer, serverResponse{mesgResourceNotFound}, http.StatusNotFound)
		return
	}

	origins := set.StringSlice(originSet)
	sort.Strings(origins)

	response := SourceResponse{
		Name:    sourceName,
		Origins: origins,
		Updated: server.Catalog.Updated.Format(time.RFC3339),
	}

	server.serveResponse(writer, response, http.StatusOK)
}

func (server *Server) serveSourceList(writer http.ResponseWriter, request *http.Request) {
	var offset, limit int

	if response, status := server.parseListRequest(writer, request, &offset, &limit); status != http.StatusOK {
		server.serveResponse(writer, response, status)
		return
	}

	originName := request.FormValue("origin")

	sourceSet := set.New(set.ThreadSafe)

	for _, origin := range server.Catalog.Origins {
		if originName != "" && origin.Name != originName {
			continue
		}

		for key := range origin.Sources {
			if request.FormValue("filter") != "" && !utils.FilterMatch(request.FormValue("filter"), key) {
				continue
			}

			sourceSet.Add(key)
		}
	}

	response := &listResponse{
		list:   StringListResponse(set.StringSlice(sourceSet)),
		offset: offset,
		limit:  limit,
	}

	server.applyResponseLimit(writer, request, response)

	server.serveResponse(writer, response.list, http.StatusOK)
}

func (server *Server) serveMetric(writer http.ResponseWriter, request *http.Request) {
	metricName := strings.TrimPrefix(request.URL.Path, urlCatalogPath+"metrics/")

	if metricName == "" {
		server.serveMetricList(writer, request)
		return
	} else if response, status := server.parseShowRequest(writer, request); status != http.StatusOK {
		server.serveResponse(writer, response, status)
		return
	}

	originSet := set.New(set.ThreadSafe)
	sourceSet := set.New(set.ThreadSafe)

	for _, origin := range server.Catalog.Origins {
		for _, source := range origin.Sources {
			if _, ok := source.Metrics[metricName]; ok {
				originSet.Add(origin.Name)
				sourceSet.Add(source.Name)
			}
		}
	}

	if originSet.Size() == 0 {
		server.serveResponse(writer, serverResponse{mesgResourceNotFound}, http.StatusNotFound)
		return
	}

	origins := set.StringSlice(originSet)
	sort.Strings(origins)

	sources := set.StringSlice(sourceSet)
	sort.Strings(sources)

	response := MetricResponse{
		Name:    metricName,
		Origins: origins,
		Sources: sources,
		Updated: server.Catalog.Updated.Format(time.RFC3339),
	}

	server.serveResponse(writer, response, http.StatusOK)
}

func (server *Server) serveMetricList(writer http.ResponseWriter, request *http.Request) {
	var offset, limit int

	if response, status := server.parseListRequest(writer, request, &offset, &limit); status != http.StatusOK {
		server.serveResponse(writer, response, status)
		return
	}

	originName := request.FormValue("origin")
	sourceName := request.FormValue("source")

	sourceSet := set.New(set.ThreadSafe)

	if strings.HasPrefix(sourceName, library.LibraryGroupPrefix) {
		for _, entryName := range server.Library.ExpandGroup(
			strings.TrimPrefix(sourceName, library.LibraryGroupPrefix),
			library.LibraryItemSourceGroup,
		) {
			sourceSet.Add(entryName)
		}
	} else if sourceName != "" {
		sourceSet.Add(sourceName)
	}

	metricSet := set.New(set.ThreadSafe)

	for _, origin := range server.Catalog.Origins {
		if originName != "" && origin.Name != originName {
			continue
		}

		for _, source := range origin.Sources {
			if sourceName != "" && sourceSet.IsEmpty() || !sourceSet.IsEmpty() && !sourceSet.Has(source.Name) {
				continue
			}

			for key := range source.Metrics {
				if request.FormValue("filter") != "" && !utils.FilterMatch(request.FormValue("filter"), key) {
					continue
				}

				metricSet.Add(key)
			}
		}
	}

	response := &listResponse{
		list:   StringListResponse(set.StringSlice(metricSet)),
		offset: offset,
		limit:  limit,
	}

	server.applyResponseLimit(writer, request, response)

	server.serveResponse(writer, response.list, http.StatusOK)
}
