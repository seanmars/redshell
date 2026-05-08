export namespace agent {
	
	export class Agent {
	    id: string;
	    label: string;
	    configDir: string;
	    settingsFile: string;
	    version: string;
	    configured: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Agent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.configDir = source["configDir"];
	        this.settingsFile = source["settingsFile"];
	        this.version = source["version"];
	        this.configured = source["configured"];
	    }
	}
	export class SetupState {
	    enabledAgents: string[];
	    agentSetupCompleted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SetupState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabledAgents = source["enabledAgents"];
	        this.agentSetupCompleted = source["agentSetupCompleted"];
	    }
	}

}

export namespace app {
	
	export class RefreshResult {
	    refreshed: string[];
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new RefreshResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.refreshed = source["refreshed"];
	        this.errors = source["errors"];
	    }
	}

}

export namespace hooks {
	
	export class DisableFlag {
	    sourceID: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new DisableFlag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceID = source["sourceID"];
	        this.path = source["path"];
	    }
	}
	export class Hook {
	    id: string;
	    sourceID: string;
	    event: string;
	    matcher?: string;
	    type: string;
	    summary: string;
	    dupCount: number;
	    raw: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new Hook(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceID = source["sourceID"];
	        this.event = source["event"];
	        this.matcher = source["matcher"];
	        this.type = source["type"];
	        this.summary = source["summary"];
	        this.dupCount = source["dupCount"];
	        this.raw = source["raw"];
	    }
	}
	export class ListOpts {
	    workspace: string;
	
	    static createFrom(source: any = {}) {
	        return new ListOpts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = source["workspace"];
	    }
	}
	export class SourceError {
	    sourceID: string;
	    path: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SourceError(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sourceID = source["sourceID"];
	        this.path = source["path"];
	        this.message = source["message"];
	    }
	}
	export class Source {
	    id: string;
	    kind: string;
	    path: string;
	    label: string;
	    pluginKey?: string;
	    scope?: string;
	
	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.label = source["label"];
	        this.pluginKey = source["pluginKey"];
	        this.scope = source["scope"];
	    }
	}
	export class Listing {
	    agentID: string;
	    sources: Source[];
	    hooks: Hook[];
	    errors: SourceError[];
	    disableAll: DisableFlag[];
	    emptyReason: string;
	
	    static createFrom(source: any = {}) {
	        return new Listing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agentID = source["agentID"];
	        this.sources = this.convertValues(source["sources"], Source);
	        this.hooks = this.convertValues(source["hooks"], Hook);
	        this.errors = this.convertValues(source["errors"], SourceError);
	        this.disableAll = this.convertValues(source["disableAll"], DisableFlag);
	        this.emptyReason = source["emptyReason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace marketplace {
	
	export class Marketplace {
	    id: string;
	    url: string;
	    name?: Record<string, string>;
	    addedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Marketplace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.name = source["name"];
	        this.addedAt = source["addedAt"];
	    }
	}

}

export namespace plugin {
	
	export class AgentUpdateOutcome {
	    agentId: string;
	    ok: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new AgentUpdateOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agentId = source["agentId"];
	        this.ok = source["ok"];
	        this.error = source["error"];
	    }
	}
	export class MarketplacePlugin {
	    name: string;
	    project: string;
	    marketplace: string;
	    marketplaceName: string;
	    installName: string;
	    description?: string;
	    agent: string;
	
	    static createFrom(source: any = {}) {
	        return new MarketplacePlugin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.project = source["project"];
	        this.marketplace = source["marketplace"];
	        this.marketplaceName = source["marketplaceName"];
	        this.installName = source["installName"];
	        this.description = source["description"];
	        this.agent = source["agent"];
	    }
	}
	export class FetchAllResult {
	    plugins: MarketplacePlugin[];
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new FetchAllResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plugins = this.convertValues(source["plugins"], MarketplacePlugin);
	        this.errors = source["errors"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class InstalledPlugin {
	    displayName: string;
	    uninstallName: string;
	    agent: string;
	    marketplaceName: string;
	
	    static createFrom(source: any = {}) {
	        return new InstalledPlugin(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayName = source["displayName"];
	        this.uninstallName = source["uninstallName"];
	        this.agent = source["agent"];
	        this.marketplaceName = source["marketplaceName"];
	    }
	}
	
	export class UpdateAgentMarketplacesResult {
	    outcomes: AgentUpdateOutcome[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateAgentMarketplacesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.outcomes = this.convertValues(source["outcomes"], AgentUpdateOutcome);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace preferences {
	
	export class AutoUpdate {
	    enabled: boolean;
	    intervalHours: number;
	    source: string;
	    githubRepo: string;
	    gitlabHost: string;
	    gitlabProject: string;
	    skipVersion: string;
	    lastCheckedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AutoUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.intervalHours = source["intervalHours"];
	        this.source = source["source"];
	        this.githubRepo = source["githubRepo"];
	        this.gitlabHost = source["gitlabHost"];
	        this.gitlabProject = source["gitlabProject"];
	        this.skipVersion = source["skipVersion"];
	        this.lastCheckedAt = source["lastCheckedAt"];
	    }
	}

}

export namespace sessionhistory {
	
	export class Event {
	    index: number;
	    kind: string;
	    subtype?: string;
	    summary: string;
	    raw: Record<string, any>;
	    children?: Event[];
	
	    static createFrom(source: any = {}) {
	        return new Event(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.kind = source["kind"];
	        this.subtype = source["subtype"];
	        this.summary = source["summary"];
	        this.raw = source["raw"];
	        this.children = this.convertValues(source["children"], Event);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EventPage {
	    agentID: string;
	    sessionID: string;
	    offset: number;
	    limit: number;
	    total: number;
	    hasMore: boolean;
	    skippedLines: number;
	    events: Event[];
	
	    static createFrom(source: any = {}) {
	        return new EventPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agentID = source["agentID"];
	        this.sessionID = source["sessionID"];
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	        this.total = source["total"];
	        this.hasMore = source["hasMore"];
	        this.skippedLines = source["skippedLines"];
	        this.events = this.convertValues(source["events"], Event);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionMeta {
	    agentID: string;
	    sessionID: string;
	    displayName?: string;
	    cwd?: string;
	    repository?: string;
	    branch?: string;
	    summary?: string;
	    createdAt?: string;
	    updatedAt?: string;
	    modifiedAt?: string;
	    byteSize?: number;
	    hasEvents: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SessionMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agentID = source["agentID"];
	        this.sessionID = source["sessionID"];
	        this.displayName = source["displayName"];
	        this.cwd = source["cwd"];
	        this.repository = source["repository"];
	        this.branch = source["branch"];
	        this.summary = source["summary"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.modifiedAt = source["modifiedAt"];
	        this.byteSize = source["byteSize"];
	        this.hasEvents = source["hasEvents"];
	    }
	}
	export class SessionGroup {
	    encodedDir: string;
	    cwd?: string;
	    sessions: SessionMeta[];
	
	    static createFrom(source: any = {}) {
	        return new SessionGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.encodedDir = source["encodedDir"];
	        this.cwd = source["cwd"];
	        this.sessions = this.convertValues(source["sessions"], SessionMeta);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Listing {
	    agentID: string;
	    kind: string;
	    groups?: SessionGroup[];
	    flat?: SessionMeta[];
	
	    static createFrom(source: any = {}) {
	        return new Listing(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.agentID = source["agentID"];
	        this.kind = source["kind"];
	        this.groups = this.convertValues(source["groups"], SessionGroup);
	        this.flat = this.convertValues(source["flat"], SessionMeta);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace updater {
	
	export class Release {
	    version: string;
	    // Go type: time
	    publishedAt: any;
	    notes: string;
	    assetUrl: string;
	    assetName: string;
	    assetSize: number;
	    checksumsUrl: string;
	    installerAssetUrl?: string;
	    installerAssetName?: string;
	    installerAssetSize?: number;
	
	    static createFrom(source: any = {}) {
	        return new Release(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.publishedAt = this.convertValues(source["publishedAt"], null);
	        this.notes = source["notes"];
	        this.assetUrl = source["assetUrl"];
	        this.assetName = source["assetName"];
	        this.assetSize = source["assetSize"];
	        this.checksumsUrl = source["checksumsUrl"];
	        this.installerAssetUrl = source["installerAssetUrl"];
	        this.installerAssetName = source["installerAssetName"];
	        this.installerAssetSize = source["installerAssetSize"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PeekResult {
	    github?: Release;
	    gitlab?: Release;
	    errors?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new PeekResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.github = this.convertValues(source["github"], Release);
	        this.gitlab = this.convertValues(source["gitlab"], Release);
	        this.errors = source["errors"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class State {
	    enabled: boolean;
	    source: string;
	    intervalHours: number;
	    runningVersion: string;
	    lastCheckedAt: string;
	    latestAvailable?: Release;
	    skipVersion: string;
	    inProgress: boolean;
	    manualRequired: boolean;
	    buildKind: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.source = source["source"];
	        this.intervalHours = source["intervalHours"];
	        this.runningVersion = source["runningVersion"];
	        this.lastCheckedAt = source["lastCheckedAt"];
	        this.latestAvailable = this.convertValues(source["latestAvailable"], Release);
	        this.skipVersion = source["skipVersion"];
	        this.inProgress = source["inProgress"];
	        this.manualRequired = source["manualRequired"];
	        this.buildKind = source["buildKind"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

