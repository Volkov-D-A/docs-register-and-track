export namespace dto {
	
	export class AccessSections {
	    dashboard: boolean;
	    incoming: boolean;
	    outgoing: boolean;
	    appeals: boolean;
	    orders: boolean;
	    controls: boolean;
	    references: boolean;
	    statistics: boolean;
	    reports: boolean;
	    settings: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AccessSections(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dashboard = source["dashboard"];
	        this.incoming = source["incoming"];
	        this.outgoing = source["outgoing"];
	        this.appeals = source["appeals"];
	        this.orders = source["orders"];
	        this.controls = source["controls"];
	        this.references = source["references"];
	        this.statistics = source["statistics"];
	        this.reports = source["reports"];
	        this.settings = source["settings"];
	    }
	}
	export class AdminAuditLog {
	    id: string;
	    userName: string;
	    action: string;
	    details: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AdminAuditLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.userName = source["userName"];
	        this.action = source["action"];
	        this.details = source["details"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class AdminAuditLogPage {
	    items: AdminAuditLog[];
	    total: number;
	    page: number;
	
	    static createFrom(source: any = {}) {
	        return new AdminAuditLogPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], AdminAuditLog);
	        this.total = source["total"];
	        this.page = source["page"];
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
	export class AdminNumberOverrideRequest {
	    mode: string;
	    number: number;
	    suffix: string;
	
	    static createFrom(source: any = {}) {
	        return new AdminNumberOverrideRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.number = source["number"];
	        this.suffix = source["suffix"];
	    }
	}
	export class AdminDraftCreateRequest {
	    nomenclatureId: string;
	    registrationDate: string;
	    adminNumberOverride?: AdminNumberOverrideRequest;
	    idempotencyKey?: string;
	
	    static createFrom(source: any = {}) {
	        return new AdminDraftCreateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nomenclatureId = source["nomenclatureId"];
	        this.registrationDate = source["registrationDate"];
	        this.adminNumberOverride = this.convertValues(source["adminNumberOverride"], AdminNumberOverrideRequest);
	        this.idempotencyKey = source["idempotencyKey"];
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
	
	export class AdministrativeOrderAcknowledgmentPerson {
	    id: string;
	    fullName: string;
	    // Go type: time
	    acknowledgedAt?: any;
	    acknowledgedByName?: string;
	    position: number;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AdministrativeOrderAcknowledgmentPerson(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fullName = source["fullName"];
	        this.acknowledgedAt = this.convertValues(source["acknowledgedAt"], null);
	        this.acknowledgedByName = source["acknowledgedByName"];
	        this.position = source["position"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class AdministrativeOrderDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    orderNumber: string;
	    // Go type: time
	    orderDate: any;
	    title: string;
	    documentTypeId: string;
	    content: string;
	    pagesCount: number;
	    executionController: string;
	    // Go type: time
	    executionDeadline?: any;
	    isActive: boolean;
	    // Go type: time
	    cancelledAt?: any;
	    acknowledgmentPeople?: AdministrativeOrderAcknowledgmentPerson[];
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new AdministrativeOrderDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.orderNumber = source["orderNumber"];
	        this.orderDate = this.convertValues(source["orderDate"], null);
	        this.title = source["title"];
	        this.documentTypeId = source["documentTypeId"];
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
	        this.executionController = source["executionController"];
	        this.executionDeadline = this.convertValues(source["executionDeadline"], null);
	        this.isActive = source["isActive"];
	        this.cancelledAt = this.convertValues(source["cancelledAt"], null);
	        this.acknowledgmentPeople = this.convertValues(source["acknowledgmentPeople"], AdministrativeOrderAcknowledgmentPerson);
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class SystemSchemaStatus {
	    currentVersion: number;
	    requiredVersion: number;
	    compatible: boolean;
	    dirty: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SystemSchemaStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.requiredVersion = source["requiredVersion"];
	        this.compatible = source["compatible"];
	        this.dirty = source["dirty"];
	    }
	}
	export class SystemStatus {
	    status: string;
	    code: string;
	    apiVersion: string;
	    serverVersion: string;
	    maintenance: boolean;
	    schema: SystemSchemaStatus;
	
	    static createFrom(source: any = {}) {
	        return new SystemStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.code = source["code"];
	        this.apiVersion = source["apiVersion"];
	        this.serverVersion = source["serverVersion"];
	        this.maintenance = source["maintenance"];
	        this.schema = this.convertValues(source["schema"], SystemSchemaStatus);
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
	export class BuildIdentity {
	    number: string;
	    revision: string;
	    fingerprint?: string;
	
	    static createFrom(source: any = {}) {
	        return new BuildIdentity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.number = source["number"];
	        this.revision = source["revision"];
	        this.fingerprint = source["fingerprint"];
	    }
	}
	export class CompatibilityResult {
	    buildProtocol: number;
	    serverBuild: BuildIdentity;
	    compatible: boolean;
	    code: string;
	    apiVersion: string;
	    serverVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new CompatibilityResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.buildProtocol = source["buildProtocol"];
	        this.serverBuild = this.convertValues(source["serverBuild"], BuildIdentity);
	        this.compatible = source["compatible"];
	        this.code = source["code"];
	        this.apiVersion = source["apiVersion"];
	        this.serverVersion = source["serverVersion"];
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
	export class BootstrapStatus {
	    state: string;
	    code: string;
	    message: string;
	    compatibility?: CompatibilityResult;
	    system?: SystemStatus;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.code = source["code"];
	        this.message = source["message"];
	        this.compatibility = this.convertValues(source["compatibility"], CompatibilityResult);
	        this.system = this.convertValues(source["system"], SystemStatus);
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
	
	export class DocumentResolution {
	    id: string;
	    resolution?: string;
	    resolutionAuthor?: string;
	    resolutionExecutors?: string;
	    position: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentResolution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.resolution = source["resolution"];
	        this.resolutionAuthor = source["resolutionAuthor"];
	        this.resolutionExecutors = source["resolutionExecutors"];
	        this.position = source["position"];
	    }
	}
	export class DocumentCorrespondentRegistration {
	    id: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    correspondentName?: string;
	    position: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentCorrespondentRegistration(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.correspondentName = source["correspondentName"];
	        this.position = source["position"];
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
	export class CitizenAppealDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    // Go type: time
	    appealDate: any;
	    documentTypeId: string;
	    content: string;
	    pagesCount: number;
	    applicantFullName: string;
	    registrationAddress: string;
	    appealType: string;
	    applicantCategory: string;
	    attachmentPagesCount: number;
	    hasEnvelope: boolean;
	    receivedFromPos: boolean;
	    correspondents?: DocumentCorrespondentRegistration[];
	    resolutions?: DocumentResolution[];
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new CitizenAppealDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.appealDate = this.convertValues(source["appealDate"], null);
	        this.documentTypeId = source["documentTypeId"];
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
	        this.applicantFullName = source["applicantFullName"];
	        this.registrationAddress = source["registrationAddress"];
	        this.appealType = source["appealType"];
	        this.applicantCategory = source["applicantCategory"];
	        this.attachmentPagesCount = source["attachmentPagesCount"];
	        this.hasEnvelope = source["hasEnvelope"];
	        this.receivedFromPos = source["receivedFromPos"];
	        this.correspondents = this.convertValues(source["correspondents"], DocumentCorrespondentRegistration);
	        this.resolutions = this.convertValues(source["resolutions"], DocumentResolution);
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	
	export class DocumentKindAccessSummary {
	    code: string;
	    name: string;
	    availableActions: string[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentKindAccessSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.availableActions = source["availableActions"];
	    }
	}
	export class CurrentAccessSummary {
	    sections: AccessSections;
	    documentKinds: DocumentKindAccessSummary[];
	    systemPermissions: string[];
	
	    static createFrom(source: any = {}) {
	        return new CurrentAccessSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sections = this.convertValues(source["sections"], AccessSections);
	        this.documentKinds = this.convertValues(source["documentKinds"], DocumentKindAccessSummary);
	        this.systemPermissions = source["systemPermissions"];
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
	export class Department {
	    id: string;
	    name: string;
	    nomenclatureIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new Department(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.nomenclatureIds = source["nomenclatureIds"];
	    }
	}
	export class OutgoingDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    outgoingNumber: string;
	    // Go type: time
	    outgoingDate: any;
	    documentTypeId: string;
	    content: string;
	    pagesCount: number;
	    attachmentPagesCount: number;
	    senderSignatory: string;
	    senderExecutor: string;
	    recipientOrgName?: string;
	    addressee: string;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new OutgoingDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.outgoingNumber = source["outgoingNumber"];
	        this.outgoingDate = this.convertValues(source["outgoingDate"], null);
	        this.documentTypeId = source["documentTypeId"];
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
	        this.attachmentPagesCount = source["attachmentPagesCount"];
	        this.senderSignatory = source["senderSignatory"];
	        this.senderExecutor = source["senderExecutor"];
	        this.recipientOrgName = source["recipientOrgName"];
	        this.addressee = source["addressee"];
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class IncomingDocument {
	    id: string;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    incomingNumber: string;
	    // Go type: time
	    incomingDate: any;
	    documentTypeId: string;
	    content: string;
	    pagesCount: number;
	    attachmentPagesCount: number;
	    correspondents?: DocumentCorrespondentRegistration[];
	    senderSignatory: string;
	    resolution?: string;
	    resolutionAuthor?: string;
	    resolutionExecutors?: string;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new IncomingDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.incomingNumber = source["incomingNumber"];
	        this.incomingDate = this.convertValues(source["incomingDate"], null);
	        this.documentTypeId = source["documentTypeId"];
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
	        this.attachmentPagesCount = source["attachmentPagesCount"];
	        this.correspondents = this.convertValues(source["correspondents"], DocumentCorrespondentRegistration);
	        this.senderSignatory = source["senderSignatory"];
	        this.resolution = source["resolution"];
	        this.resolutionAuthor = source["resolutionAuthor"];
	        this.resolutionExecutors = source["resolutionExecutors"];
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class DocumentCard {
	    id: string;
	    kindCode: string;
	    kindName: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    documentTypeId: string;
	    content: string;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    incomingLetter?: IncomingDocument;
	    outgoingLetter?: OutgoingDocument;
	    citizenAppeal?: CitizenAppealDocument;
	    administrativeOrder?: AdministrativeOrderDocument;
	
	    static createFrom(source: any = {}) {
	        return new DocumentCard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kindCode = source["kindCode"];
	        this.kindName = source["kindName"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.documentTypeId = source["documentTypeId"];
	        this.content = source["content"];
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.incomingLetter = this.convertValues(source["incomingLetter"], IncomingDocument);
	        this.outgoingLetter = this.convertValues(source["outgoingLetter"], OutgoingDocument);
	        this.citizenAppeal = this.convertValues(source["citizenAppeal"], CitizenAppealDocument);
	        this.administrativeOrder = this.convertValues(source["administrativeOrder"], AdministrativeOrderDocument);
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
	
	
	export class DocumentLink {
	    id: string;
	    sourceKind: string;
	    sourceId: string;
	    targetKind: string;
	    linkType: string;
	    sourceNumber?: string;
	    targetNumber?: string;
	    sourceSubject?: string;
	    targetSubject?: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sourceKind = source["sourceKind"];
	        this.sourceId = source["sourceId"];
	        this.targetKind = source["targetKind"];
	        this.linkType = source["linkType"];
	        this.sourceNumber = source["sourceNumber"];
	        this.targetNumber = source["targetNumber"];
	        this.sourceSubject = source["sourceSubject"];
	        this.targetSubject = source["targetSubject"];
	    }
	}
	export class DocumentListItem {
	    id: string;
	    kindCode: string;
	    kindName: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    nomenclatureId: string;
	    nomenclatureName?: string;
	    documentTypeId: string;
	    content: string;
	    pagesCount: number;
	    createdByName?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    incomingNumber?: string;
	    // Go type: time
	    incomingDate?: any;
	    // Go type: time
	    appealDate?: any;
	    outgoingNumber?: string;
	    // Go type: time
	    outgoingDate?: any;
	    correspondents?: DocumentCorrespondentRegistration[];
	    senderSignatory?: string;
	    resolution?: string;
	    resolutionAuthor?: string;
	    resolutionExecutors?: string;
	    resolutions?: DocumentResolution[];
	    recipientOrgName?: string;
	    addressee?: string;
	    senderExecutor?: string;
	    applicantFullName?: string;
	    registrationAddress?: string;
	    appealType?: string;
	    applicantCategory?: string;
	    attachmentPagesCount?: number;
	    hasEnvelope?: boolean;
	    receivedFromPos?: boolean;
	    orderNumber?: string;
	    // Go type: time
	    orderDate?: any;
	    title?: string;
	    executionController?: string;
	    // Go type: time
	    executionDeadline?: any;
	    isActive: boolean;
	    // Go type: time
	    cancelledAt?: any;
	    pendingAcknowledgmentsCount?: number;
	    acknowledgmentPeople?: AdministrativeOrderAcknowledgmentPerson[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kindCode = source["kindCode"];
	        this.kindName = source["kindName"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureName = source["nomenclatureName"];
	        this.documentTypeId = source["documentTypeId"];
	        this.content = source["content"];
	        this.pagesCount = source["pagesCount"];
	        this.createdByName = source["createdByName"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.incomingNumber = source["incomingNumber"];
	        this.incomingDate = this.convertValues(source["incomingDate"], null);
	        this.appealDate = this.convertValues(source["appealDate"], null);
	        this.outgoingNumber = source["outgoingNumber"];
	        this.outgoingDate = this.convertValues(source["outgoingDate"], null);
	        this.correspondents = this.convertValues(source["correspondents"], DocumentCorrespondentRegistration);
	        this.senderSignatory = source["senderSignatory"];
	        this.resolution = source["resolution"];
	        this.resolutionAuthor = source["resolutionAuthor"];
	        this.resolutionExecutors = source["resolutionExecutors"];
	        this.resolutions = this.convertValues(source["resolutions"], DocumentResolution);
	        this.recipientOrgName = source["recipientOrgName"];
	        this.addressee = source["addressee"];
	        this.senderExecutor = source["senderExecutor"];
	        this.applicantFullName = source["applicantFullName"];
	        this.registrationAddress = source["registrationAddress"];
	        this.appealType = source["appealType"];
	        this.applicantCategory = source["applicantCategory"];
	        this.attachmentPagesCount = source["attachmentPagesCount"];
	        this.hasEnvelope = source["hasEnvelope"];
	        this.receivedFromPos = source["receivedFromPos"];
	        this.orderNumber = source["orderNumber"];
	        this.orderDate = this.convertValues(source["orderDate"], null);
	        this.title = source["title"];
	        this.executionController = source["executionController"];
	        this.executionDeadline = this.convertValues(source["executionDeadline"], null);
	        this.isActive = source["isActive"];
	        this.cancelledAt = this.convertValues(source["cancelledAt"], null);
	        this.pendingAcknowledgmentsCount = source["pendingAcknowledgmentsCount"];
	        this.acknowledgmentPeople = this.convertValues(source["acknowledgmentPeople"], AdministrativeOrderAcknowledgmentPerson);
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
	
	export class DocumentSearchItem {
	    id: string;
	    kindCode: string;
	    registrationNumber: string;
	    // Go type: time
	    registrationDate: any;
	    content: string;
	    resolution: string;
	    correspondent: string;
	    person: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentSearchItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kindCode = source["kindCode"];
	        this.registrationNumber = source["registrationNumber"];
	        this.registrationDate = this.convertValues(source["registrationDate"], null);
	        this.content = source["content"];
	        this.resolution = source["resolution"];
	        this.correspondent = source["correspondent"];
	        this.person = source["person"];
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
	export class DocumentSearchRequest {
	    query: string;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentSearchRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class DocumentSearchResult {
	    items: DocumentSearchItem[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentSearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], DocumentSearchItem);
	        this.totalCount = source["totalCount"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
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
	
	export class JournalEntry {
	    id: string;
	    userName?: string;
	    action: string;
	    details: string;
	    // Go type: time
	    createdAt: any;
	
	    static createFrom(source: any = {}) {
	        return new JournalEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.userName = source["userName"];
	        this.action = source["action"];
	        this.details = source["details"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
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
	export class MigrationStatus {
	    currentVersion: number;
	    dirty: boolean;
	    availableCount: number;
	    latestAvailableVersion: number;
	    upToDate: boolean;
	    schemaTooNew: boolean;
	    compatible: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MigrationStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.dirty = source["dirty"];
	        this.availableCount = source["availableCount"];
	        this.latestAvailableVersion = source["latestAvailableVersion"];
	        this.upToDate = source["upToDate"];
	        this.schemaTooNew = source["schemaTooNew"];
	        this.compatible = source["compatible"];
	    }
	}
	export class Nomenclature {
	    id: string;
	    name: string;
	    index: string;
	    year: number;
	    kindCode: string;
	    separator: string;
	    numberingMode: string;
	    nextNumber: number;
	    isActive: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Nomenclature(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.index = source["index"];
	        this.year = source["year"];
	        this.kindCode = source["kindCode"];
	        this.separator = source["separator"];
	        this.numberingMode = source["numberingMode"];
	        this.nextNumber = source["nextNumber"];
	        this.isActive = source["isActive"];
	    }
	}
	export class Organization {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new Organization(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	
	export class PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_DocumentListItem_ {
	    items: DocumentListItem[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	    nextCursor?: string;
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_DocumentListItem_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], DocumentListItem);
	        this.totalCount = source["totalCount"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.nextCursor = source["nextCursor"];
	        this.hasMore = source["hasMore"];
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
	export class UserEvent {
	    id: string;
	    documentId: string;
	    documentKind: string;
	    documentNumber?: string;
	    // Go type: time
	    documentDate?: any;
	    entityType: string;
	    eventType: string;
	    title: string;
	    message: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    readAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new UserEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentId = source["documentId"];
	        this.documentKind = source["documentKind"];
	        this.documentNumber = source["documentNumber"];
	        this.documentDate = this.convertValues(source["documentDate"], null);
	        this.entityType = source["entityType"];
	        this.eventType = source["eventType"];
	        this.title = source["title"];
	        this.message = source["message"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.readAt = this.convertValues(source["readAt"], null);
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
	export class PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_UserEvent_ {
	    items: UserEvent[];
	    totalCount: number;
	    page: number;
	    pageSize: number;
	    nextCursor?: string;
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PagedResult_github_com_Volkov_D_A_docs_register_and_track_internal_dto_UserEvent_(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], UserEvent);
	        this.totalCount = source["totalCount"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.nextCursor = source["nextCursor"];
	        this.hasMore = source["hasMore"];
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
	export class ResolutionExecutor {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new ResolutionExecutor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	
	
	export class User {
	    id: string;
	    login: string;
	    fullName: string;
	    lastName: string;
	    firstName: string;
	    patronymic: string;
	    noPatronymic: boolean;
	    isDocumentParticipant: boolean;
	    isActive: boolean;
	    failedLoginAttempts: number;
	    temporaryPassword?: string;
	    systemPermissions: string[];
	    department?: Department;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.login = source["login"];
	        this.fullName = source["fullName"];
	        this.lastName = source["lastName"];
	        this.firstName = source["firstName"];
	        this.patronymic = source["patronymic"];
	        this.noPatronymic = source["noPatronymic"];
	        this.isDocumentParticipant = source["isDocumentParticipant"];
	        this.isActive = source["isActive"];
	        this.failedLoginAttempts = source["failedLoginAttempts"];
	        this.temporaryPassword = source["temporaryPassword"];
	        this.systemPermissions = source["systemPermissions"];
	        this.department = this.convertValues(source["department"], Department);
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
	
	export class WorkspaceDocument {
	    id: string;
	    documentKind: string;
	    documentNumber: string;
	    // Go type: time
	    documentDate: any;
	    // Go type: time
	    registeredAt: any;
	    description: string;
	    correspondents: string[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDocument(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentKind = source["documentKind"];
	        this.documentNumber = source["documentNumber"];
	        this.documentDate = this.convertValues(source["documentDate"], null);
	        this.registeredAt = this.convertValues(source["registeredAt"], null);
	        this.description = source["description"];
	        this.correspondents = source["correspondents"];
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
	export class WorkspaceDocuments {
	    available: boolean;
	    items: WorkspaceDocument[];
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceDocuments(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.items = this.convertValues(source["items"], WorkspaceDocument);
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

export namespace models {
	
	export class BackupCopy {
	    restoreConfirmation: string;
	    deleteConfirmation: string;
	    id: string;
	    format: number;
	    // Go type: time
	    createdAt: any;
	    size: number;
	    sha256: string;
	    verification: string;
	    issue: string;
	    canDelete: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupCopy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.restoreConfirmation = source["restoreConfirmation"];
	        this.deleteConfirmation = source["deleteConfirmation"];
	        this.id = source["id"];
	        this.format = source["format"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.size = source["size"];
	        this.sha256 = source["sha256"];
	        this.verification = source["verification"];
	        this.issue = source["issue"];
	        this.canDelete = source["canDelete"];
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
	export class BackupStage {
	    state: string;
	    // Go type: time
	    startedAt: any;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupStage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.error = source["error"];
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
	export class BackupJob {
	    stages: BackupStage[];
	    kind: string;
	    copyId: string;
	    id: string;
	    state: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    error?: string;
	    archiveSize: number;
	
	    static createFrom(source: any = {}) {
	        return new BackupJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stages = this.convertValues(source["stages"], BackupStage);
	        this.kind = source["kind"];
	        this.copyId = source["copyId"];
	        this.id = source["id"];
	        this.state = source["state"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.error = source["error"];
	        this.archiveSize = source["archiveSize"];
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
	export class BackupOperation {
	    stages: BackupStage[];
	    id: string;
	    copyId: string;
	    kind: string;
	    state: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    error: string;
	    canCancel: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupOperation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stages = this.convertValues(source["stages"], BackupStage);
	        this.id = source["id"];
	        this.copyId = source["copyId"];
	        this.kind = source["kind"];
	        this.state = source["state"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.error = source["error"];
	        this.canCancel = source["canCancel"];
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
	export class BackupOperationRequest {
	    copyId: string;
	    verificationId: string;
	    confirmation: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupOperationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.copyId = source["copyId"];
	        this.verificationId = source["verificationId"];
	        this.confirmation = source["confirmation"];
	    }
	}
	export class BackupOperationStarted {
	    job: BackupOperation;
	    statusToken: string;
	    // Go type: time
	    expiresAt: any;
	
	    static createFrom(source: any = {}) {
	        return new BackupOperationStarted(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.job = this.convertValues(source["job"], BackupOperation);
	        this.statusToken = source["statusToken"];
	        this.expiresAt = this.convertValues(source["expiresAt"], null);
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
	export class SMBDestination {
	    port?: number;
	    host: string;
	    share: string;
	    directory: string;
	    user: string;
	    domain: string;
	
	    static createFrom(source: any = {}) {
	        return new SMBDestination(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.port = source["port"];
	        this.host = source["host"];
	        this.share = source["share"];
	        this.directory = source["directory"];
	        this.user = source["user"];
	        this.domain = source["domain"];
	    }
	}
	export class BackupSettings {
	    smb: SMBDestination;
	    enabled: boolean;
	    time: string;
	    timezone: string;
	    weekdays: number[];
	    retentionDays: number;
	    keepCopies: number;
	    passwordSet: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.smb = this.convertValues(source["smb"], SMBDestination);
	        this.enabled = source["enabled"];
	        this.time = source["time"];
	        this.timezone = source["timezone"];
	        this.weekdays = source["weekdays"];
	        this.retentionDays = source["retentionDays"];
	        this.keepCopies = source["keepCopies"];
	        this.passwordSet = source["passwordSet"];
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
	export class BackupSettingsResponse {
	    nextRun: string;
	    settings: BackupSettings;
	    issue: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupSettingsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nextRun = source["nextRun"];
	        this.settings = this.convertValues(source["settings"], BackupSettings);
	        this.issue = source["issue"];
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
	export class BackupSettingsUpdate {
	    settings: BackupSettings;
	    password: string;
	    clearPassword: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BackupSettingsUpdate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], BackupSettings);
	        this.password = source["password"];
	        this.clearPassword = source["clearPassword"];
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
	
	export class CancelDeadlineRequest {
	    version: number;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new CancelDeadlineRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.reason = source["reason"];
	    }
	}
	export class CompleteDeadlineRequest {
	    version: number;
	    executedOn: string;
	    comment: string;
	
	    static createFrom(source: any = {}) {
	        return new CompleteDeadlineRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.executedOn = source["executedOn"];
	        this.comment = source["comment"];
	    }
	}
	export class CreateUserRequest {
	    login: string;
	    password: string;
	    lastName: string;
	    firstName: string;
	    patronymic: string;
	    noPatronymic: boolean;
	    departmentId: string;
	    isDocumentParticipant: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CreateUserRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.login = source["login"];
	        this.password = source["password"];
	        this.lastName = source["lastName"];
	        this.firstName = source["firstName"];
	        this.patronymic = source["patronymic"];
	        this.noPatronymic = source["noPatronymic"];
	        this.departmentId = source["departmentId"];
	        this.isDocumentParticipant = source["isDocumentParticipant"];
	    }
	}
	export class DeadlineControlFilter {
	    manageOnly: boolean;
	    seriesStoppedOnly: boolean;
	    documentId: string;
	    seriesId: string;
	    mineOnly: boolean;
	    status: string;
	    overdueOnly: boolean;
	    dueFrom: string;
	    dueTo: string;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineControlFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.manageOnly = source["manageOnly"];
	        this.seriesStoppedOnly = source["seriesStoppedOnly"];
	        this.documentId = source["documentId"];
	        this.seriesId = source["seriesId"];
	        this.mineOnly = source["mineOnly"];
	        this.status = source["status"];
	        this.overdueOnly = source["overdueOnly"];
	        this.dueFrom = source["dueFrom"];
	        this.dueTo = source["dueTo"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}
	export class DocumentDeadlineControl {
	    id: string;
	    documentId: string;
	    documentKind: string;
	    documentNumber: string;
	    createdBy: string;
	    executorId: string;
	    coExecutorIds: string[];
	    subject: string;
	    dueDate: string;
	    status: string;
	    executedOn?: string;
	    // Go type: time
	    completedAt?: any;
	    completedBy?: string;
	    comment: string;
	    cancellationReason: string;
	    seriesId?: string;
	    iteration: number;
	    plannedDueDate?: string;
	    version: number;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    canManage: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DocumentDeadlineControl(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentId = source["documentId"];
	        this.documentKind = source["documentKind"];
	        this.documentNumber = source["documentNumber"];
	        this.createdBy = source["createdBy"];
	        this.executorId = source["executorId"];
	        this.coExecutorIds = source["coExecutorIds"];
	        this.subject = source["subject"];
	        this.dueDate = source["dueDate"];
	        this.status = source["status"];
	        this.executedOn = source["executedOn"];
	        this.completedAt = this.convertValues(source["completedAt"], null);
	        this.completedBy = source["completedBy"];
	        this.comment = source["comment"];
	        this.cancellationReason = source["cancellationReason"];
	        this.seriesId = source["seriesId"];
	        this.iteration = source["iteration"];
	        this.plannedDueDate = source["plannedDueDate"];
	        this.version = source["version"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.canManage = source["canManage"];
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
	export class DeadlineControlPage {
	    items: DocumentDeadlineControl[];
	    total: number;
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineControlPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], DocumentDeadlineControl);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
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
	export class DeadlineControlRequest {
	    documentId: string;
	    executorId: string;
	    coExecutorIds: string[];
	    subject: string;
	    dueDate: string;
	    version: number;
	    changeReason: string;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineControlRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.documentId = source["documentId"];
	        this.executorId = source["executorId"];
	        this.coExecutorIds = source["coExecutorIds"];
	        this.subject = source["subject"];
	        this.dueDate = source["dueDate"];
	        this.version = source["version"];
	        this.changeReason = source["changeReason"];
	    }
	}
	export class DeadlineExecutor {
	    id: string;
	    userId?: string;
	    name: string;
	    institution: string;
	    position: string;
	    isActive: boolean;
	    version: number;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineExecutor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.userId = source["userId"];
	        this.name = source["name"];
	        this.institution = source["institution"];
	        this.position = source["position"];
	        this.isActive = source["isActive"];
	        this.version = source["version"];
	    }
	}
	export class DeadlineMonthlyPoint {
	    month: number;
	    period: string;
	    total: number;
	    overdue: number;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineMonthlyPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.period = source["period"];
	        this.total = source["total"];
	        this.overdue = source["overdue"];
	    }
	}
	export class DeadlineRecurrence {
	    intervalUnit: string;
	    intervalValue: number;
	    dayRule: string;
	    dayOfMonth: number;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineRecurrence(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.intervalUnit = source["intervalUnit"];
	        this.intervalValue = source["intervalValue"];
	        this.dayRule = source["dayRule"];
	        this.dayOfMonth = source["dayOfMonth"];
	    }
	}
	export class DeadlineSeriesActionRequest {
	    version: number;
	    reason: string;
	    expectedNextDueDate: string;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineSeriesActionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.reason = source["reason"];
	        this.expectedNextDueDate = source["expectedNextDueDate"];
	    }
	}
	export class DeadlineSeriesRequest {
	    control: DeadlineControlRequest;
	    recurrence: DeadlineRecurrence;
	    version: number;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineSeriesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.control = this.convertValues(source["control"], DeadlineControlRequest);
	        this.recurrence = this.convertValues(source["recurrence"], DeadlineRecurrence);
	        this.version = source["version"];
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
	export class DeadlineSeriesResumePreview {
	    version: number;
	    dueDate: string;
	    createsIteration: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DeadlineSeriesResumePreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.dueDate = source["dueDate"];
	        this.createsIteration = source["createsIteration"];
	    }
	}
	export class StatisticsReportRow {
	    key: string;
	    name: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new StatisticsReportRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.count = source["count"];
	    }
	}
	export class StatisticsSeriesPoint {
	    month: number;
	    period: string;
	    categoryKey: string;
	    categoryName: string;
	    value: number;
	
	    static createFrom(source: any = {}) {
	        return new StatisticsSeriesPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.month = source["month"];
	        this.period = source["period"];
	        this.categoryKey = source["categoryKey"];
	        this.categoryName = source["categoryName"];
	        this.value = source["value"];
	    }
	}
	export class DeadlineStatistics {
	    year: number;
	    monthlyTotals: DeadlineMonthlyPoint[];
	    monthlyByExecutor: StatisticsSeriesPoint[];
	    overdueRating: StatisticsReportRow[];
	    statusCounts: StatisticsReportRow[];
	
	    static createFrom(source: any = {}) {
	        return new DeadlineStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.monthlyTotals = this.convertValues(source["monthlyTotals"], DeadlineMonthlyPoint);
	        this.monthlyByExecutor = this.convertValues(source["monthlyByExecutor"], StatisticsSeriesPoint);
	        this.overdueRating = this.convertValues(source["overdueRating"], StatisticsReportRow);
	        this.statusCounts = this.convertValues(source["statusCounts"], StatisticsReportRow);
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
	export class StatisticsOption {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new StatisticsOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class DeadlineStatisticsFilters {
	    users: StatisticsOption[];
	
	    static createFrom(source: any = {}) {
	        return new DeadlineStatisticsFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.users = this.convertValues(source["users"], StatisticsOption);
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
	export class DeadlineStatisticsReport {
	    total: number;
	    rows: StatisticsReportRow[];
	
	    static createFrom(source: any = {}) {
	        return new DeadlineStatisticsReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.rows = this.convertValues(source["rows"], StatisticsReportRow);
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
	export class DocumentAccessScope {
	    UserID: string;
	
	    static createFrom(source: any = {}) {
	        return new DocumentAccessScope(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UserID = source["UserID"];
	    }
	}
	
	export class DocumentDeadlineSeries {
	    id: string;
	    documentId: string;
	    createdBy: string;
	    executorId: string;
	    coExecutorIds: string[];
	    subject: string;
	    recurrence: DeadlineRecurrence;
	    firstDueDate: string;
	    active: boolean;
	    stopReason: string;
	    currentControlId: string;
	    currentIteration: number;
	    version: number;
	
	    static createFrom(source: any = {}) {
	        return new DocumentDeadlineSeries(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.documentId = source["documentId"];
	        this.createdBy = source["createdBy"];
	        this.executorId = source["executorId"];
	        this.coExecutorIds = source["coExecutorIds"];
	        this.subject = source["subject"];
	        this.recurrence = this.convertValues(source["recurrence"], DeadlineRecurrence);
	        this.firstDueDate = source["firstDueDate"];
	        this.active = source["active"];
	        this.stopReason = source["stopReason"];
	        this.currentControlId = source["currentControlId"];
	        this.currentIteration = source["currentIteration"];
	        this.version = source["version"];
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
	export class DocumentFilter {
	    nomenclatureId?: string;
	    nomenclatureIds?: string[];
	    kindCode?: string;
	    documentTypeId?: string;
	    orgId?: string;
	    dateFrom?: string;
	    dateTo?: string;
	    search?: string;
	    incomingNumber?: string;
	    registrationNumber?: string;
	    outgoingNumber?: string;
	    recipientName?: string;
	    senderName?: string;
	    applicantName?: string;
	    appealType?: string;
	    appealDateFrom?: string;
	    appealDateTo?: string;
	    outgoingDateFrom?: string;
	    outgoingDateTo?: string;
	    resolution?: string;
	    noResolution?: boolean;
	    orderNumber?: string;
	    executionController?: string;
	    onlyPendingAcknowledgment?: boolean;
	    orderActiveStatus?: string;
	    page: number;
	    pageSize: number;
	    cursor?: string;
	    cursorPagination?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DocumentFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nomenclatureId = source["nomenclatureId"];
	        this.nomenclatureIds = source["nomenclatureIds"];
	        this.kindCode = source["kindCode"];
	        this.documentTypeId = source["documentTypeId"];
	        this.orgId = source["orgId"];
	        this.dateFrom = source["dateFrom"];
	        this.dateTo = source["dateTo"];
	        this.search = source["search"];
	        this.incomingNumber = source["incomingNumber"];
	        this.registrationNumber = source["registrationNumber"];
	        this.outgoingNumber = source["outgoingNumber"];
	        this.recipientName = source["recipientName"];
	        this.senderName = source["senderName"];
	        this.applicantName = source["applicantName"];
	        this.appealType = source["appealType"];
	        this.appealDateFrom = source["appealDateFrom"];
	        this.appealDateTo = source["appealDateTo"];
	        this.outgoingDateFrom = source["outgoingDateFrom"];
	        this.outgoingDateTo = source["outgoingDateTo"];
	        this.resolution = source["resolution"];
	        this.noResolution = source["noResolution"];
	        this.orderNumber = source["orderNumber"];
	        this.executionController = source["executionController"];
	        this.onlyPendingAcknowledgment = source["onlyPendingAcknowledgment"];
	        this.orderActiveStatus = source["orderActiveStatus"];
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	        this.cursor = source["cursor"];
	        this.cursorPagination = source["cursorPagination"];
	    }
	}
	export class DocumentStatistics {
	    year: number;
	    totalYear: number;
	    documentsByKindMonthly: StatisticsSeriesPoint[];
	    documentsByRegistrarMonthly: StatisticsSeriesPoint[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.totalYear = source["totalYear"];
	        this.documentsByKindMonthly = this.convertValues(source["documentsByKindMonthly"], StatisticsSeriesPoint);
	        this.documentsByRegistrarMonthly = this.convertValues(source["documentsByRegistrarMonthly"], StatisticsSeriesPoint);
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
	export class DocumentStatisticsFilters {
	    kinds: StatisticsOption[];
	    nomenclature: StatisticsOption[];
	    users: StatisticsOption[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentStatisticsFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kinds = this.convertValues(source["kinds"], StatisticsOption);
	        this.nomenclature = this.convertValues(source["nomenclature"], StatisticsOption);
	        this.users = this.convertValues(source["users"], StatisticsOption);
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
	export class DocumentStatisticsReport {
	    total: number;
	    rows: StatisticsReportRow[];
	
	    static createFrom(source: any = {}) {
	        return new DocumentStatisticsReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.rows = this.convertValues(source["rows"], StatisticsReportRow);
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
	export class ExternalDeadlineExecutorRequest {
	    name: string;
	    institution: string;
	    position: string;
	    isActive: boolean;
	    version: number;
	
	    static createFrom(source: any = {}) {
	        return new ExternalDeadlineExecutorRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.institution = source["institution"];
	        this.position = source["position"];
	        this.isActive = source["isActive"];
	        this.version = source["version"];
	    }
	}
	export class FailedOutboxEvent {
	    id: number[];
	    eventType: string;
	    attempts: number;
	    lastError: string;
	    // Go type: time
	    failedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new FailedOutboxEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.eventType = source["eventType"];
	        this.attempts = source["attempts"];
	        this.lastError = source["lastError"];
	        this.failedAt = this.convertValues(source["failedAt"], null);
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
	export class GraphEdge {
	    id: string;
	    source: string;
	    target: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphEdge(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.target = source["target"];
	        this.label = source["label"];
	    }
	}
	export class GraphNode {
	    id: string;
	    label: string;
	    kindCode: string;
	    subject: string;
	    date: string;
	    sender: string;
	    recipient: string;
	    isActive?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GraphNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.kindCode = source["kindCode"];
	        this.subject = source["subject"];
	        this.date = source["date"];
	        this.sender = source["sender"];
	        this.recipient = source["recipient"];
	        this.isActive = source["isActive"];
	    }
	}
	export class GraphData {
	    nodes: GraphNode[];
	    edges: GraphEdge[];
	
	    static createFrom(source: any = {}) {
	        return new GraphData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodes = this.convertValues(source["nodes"], GraphNode);
	        this.edges = this.convertValues(source["edges"], GraphEdge);
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
	
	
	export class InitialSetupRequest {
	    password: string;
	    lastName: string;
	    firstName: string;
	    patronymic: string;
	    noPatronymic: boolean;
	
	    static createFrom(source: any = {}) {
	        return new InitialSetupRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.password = source["password"];
	        this.lastName = source["lastName"];
	        this.firstName = source["firstName"];
	        this.patronymic = source["patronymic"];
	        this.noPatronymic = source["noPatronymic"];
	    }
	}
	export class OutboxStats {
	    Pending: number;
	    Processing: number;
	    Failed: number;
	    Processed: number;
	
	    static createFrom(source: any = {}) {
	        return new OutboxStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Pending = source["Pending"];
	        this.Processing = source["Processing"];
	        this.Failed = source["Failed"];
	        this.Processed = source["Processed"];
	    }
	}
	export class ReleaseNoteChange {
	    title: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseNoteChange(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.description = source["description"];
	    }
	}
	export class ReleaseNote {
	    version: string;
	    // Go type: time
	    releasedAt: any;
	    changes: ReleaseNoteChange[];
	    isViewed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.releasedAt = this.convertValues(source["releasedAt"], null);
	        this.changes = this.convertValues(source["changes"], ReleaseNoteChange);
	        this.isViewed = source["isViewed"];
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
	
	export class ReportFilters {
	    executors: StatisticsOption[];
	    organizations: StatisticsOption[];
	    departments: StatisticsOption[];
	    kinds: StatisticsOption[];
	    nomenclature: StatisticsOption[];
	    users: StatisticsOption[];
	
	    static createFrom(source: any = {}) {
	        return new ReportFilters(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.executors = this.convertValues(source["executors"], StatisticsOption);
	        this.organizations = this.convertValues(source["organizations"], StatisticsOption);
	        this.departments = this.convertValues(source["departments"], StatisticsOption);
	        this.kinds = this.convertValues(source["kinds"], StatisticsOption);
	        this.nomenclature = this.convertValues(source["nomenclature"], StatisticsOption);
	        this.users = this.convertValues(source["users"], StatisticsOption);
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
	export class ReportRequest {
	    periodBasis?: string;
	    template: string;
	    startDate: string;
	    endDate: string;
	    organizationId?: string;
	    departmentId?: string;
	    kindCode?: string;
	    nomenclatureId?: string;
	    userId?: string;
	    status?: string;
	    timezone?: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.periodBasis = source["periodBasis"];
	        this.template = source["template"];
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.organizationId = source["organizationId"];
	        this.departmentId = source["departmentId"];
	        this.kindCode = source["kindCode"];
	        this.nomenclatureId = source["nomenclatureId"];
	        this.userId = source["userId"];
	        this.status = source["status"];
	        this.timezone = source["timezone"];
	    }
	}
	export class ReportSummary {
	    count: number;
	    incoming: number;
	    outgoing: number;
	    overdue: number;
	    openOverdue: number;
	    metric: number;
	    uniqueDocuments: number;
	
	    static createFrom(source: any = {}) {
	        return new ReportSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.count = source["count"];
	        this.incoming = source["incoming"];
	        this.outgoing = source["outgoing"];
	        this.overdue = source["overdue"];
	        this.openOverdue = source["openOverdue"];
	        this.metric = source["metric"];
	        this.uniqueDocuments = source["uniqueDocuments"];
	    }
	}
	export class ReportRow {
	    key: string;
	    name: string;
	    period?: string;
	    incoming?: number;
	    outgoing?: number;
	    overdue?: number;
	    count: number;
	    metric: number;
	
	    static createFrom(source: any = {}) {
	        return new ReportRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.name = source["name"];
	        this.period = source["period"];
	        this.incoming = source["incoming"];
	        this.outgoing = source["outgoing"];
	        this.overdue = source["overdue"];
	        this.count = source["count"];
	        this.metric = source["metric"];
	    }
	}
	export class ReportResult {
	    template: string;
	    startDate: string;
	    endDate: string;
	    generatedAt: string;
	    definition: string;
	    filters: string;
	    version: number;
	    timezone: string;
	    rows: ReportRow[];
	    summary: ReportSummary;
	
	    static createFrom(source: any = {}) {
	        return new ReportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.template = source["template"];
	        this.startDate = source["startDate"];
	        this.endDate = source["endDate"];
	        this.generatedAt = source["generatedAt"];
	        this.definition = source["definition"];
	        this.filters = source["filters"];
	        this.version = source["version"];
	        this.timezone = source["timezone"];
	        this.rows = this.convertValues(source["rows"], ReportRow);
	        this.summary = this.convertValues(source["summary"], ReportSummary);
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
	
	
	export class RollbackMigrationRequest {
	    backupCompleted: boolean;
	    backupReference: string;
	    acknowledgedDataLoss: boolean;
	    confirmation: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new RollbackMigrationRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.backupCompleted = source["backupCompleted"];
	        this.backupReference = source["backupReference"];
	        this.acknowledgedDataLoss = source["acknowledgedDataLoss"];
	        this.confirmation = source["confirmation"];
	        this.password = source["password"];
	    }
	}
	
	
	
	
	export class SystemAPIStatistics {
	    requestsSinceStart: number;
	    clientErrorsSinceStart: number;
	    serverErrorsSinceStart: number;
	    deadlineExceededSinceStart: number;
	    p95Milliseconds: number;
	    inFlight: number;
	    sampleWindow: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemAPIStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestsSinceStart = source["requestsSinceStart"];
	        this.clientErrorsSinceStart = source["clientErrorsSinceStart"];
	        this.serverErrorsSinceStart = source["serverErrorsSinceStart"];
	        this.deadlineExceededSinceStart = source["deadlineExceededSinceStart"];
	        this.p95Milliseconds = source["p95Milliseconds"];
	        this.inFlight = source["inFlight"];
	        this.sampleWindow = source["sampleWindow"];
	    }
	}
	export class SystemDatabaseStatistics {
	    poolOpen: number;
	    poolInUse: number;
	    poolIdle: number;
	    poolMax: number;
	    waitCountSinceStart: number;
	    waitMillisecondsSinceStart: number;
	    operationsSinceStart: number;
	    operationErrorsSinceStart: number;
	    operationP95Milliseconds: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemDatabaseStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.poolOpen = source["poolOpen"];
	        this.poolInUse = source["poolInUse"];
	        this.poolIdle = source["poolIdle"];
	        this.poolMax = source["poolMax"];
	        this.waitCountSinceStart = source["waitCountSinceStart"];
	        this.waitMillisecondsSinceStart = source["waitMillisecondsSinceStart"];
	        this.operationsSinceStart = source["operationsSinceStart"];
	        this.operationErrorsSinceStart = source["operationErrorsSinceStart"];
	        this.operationP95Milliseconds = source["operationP95Milliseconds"];
	    }
	}
	export class SystemOutboxStatistics {
	    pending: number;
	    processing: number;
	    failed: number;
	    processedSinceStart: number;
	    retriesSinceStart: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemOutboxStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pending = source["pending"];
	        this.processing = source["processing"];
	        this.failed = source["failed"];
	        this.processedSinceStart = source["processedSinceStart"];
	        this.retriesSinceStart = source["retriesSinceStart"];
	    }
	}
	export class SystemServiceStatistics {
	    buildVersion: string;
	    sourceRevision: string;
	    sourceDirty: boolean;
	    version: string;
	    state: string;
	    // Go type: time
	    startedAt: any;
	    uptimeSeconds: number;
	    schemaCurrentVersion: number;
	    schemaRequiredVersion: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemServiceStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.buildVersion = source["buildVersion"];
	        this.sourceRevision = source["sourceRevision"];
	        this.sourceDirty = source["sourceDirty"];
	        this.version = source["version"];
	        this.state = source["state"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.uptimeSeconds = source["uptimeSeconds"];
	        this.schemaCurrentVersion = source["schemaCurrentVersion"];
	        this.schemaRequiredVersion = source["schemaRequiredVersion"];
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
	export class SystemSetting {
	    key: string;
	    value: string;
	
	    static createFrom(source: any = {}) {
	        return new SystemSetting(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	    }
	}
	export class SystemUsageStatistics {
	    activeUsers15m: number;
	    activeSessions: number;
	
	    static createFrom(source: any = {}) {
	        return new SystemUsageStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.activeUsers15m = source["activeUsers15m"];
	        this.activeSessions = source["activeSessions"];
	    }
	}
	export class SystemStatistics {
	    clientBuildVersion: string;
	    clientRevision: string;
	    clientDirty: boolean;
	    userCount: number;
	    totalDocuments: number;
	    dbSize: string;
	    // Go type: time
	    generatedAt: any;
	    service: SystemServiceStatistics;
	    usage: SystemUsageStatistics;
	    api: SystemAPIStatistics;
	    database: SystemDatabaseStatistics;
	    outbox: SystemOutboxStatistics;
	
	    static createFrom(source: any = {}) {
	        return new SystemStatistics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clientBuildVersion = source["clientBuildVersion"];
	        this.clientRevision = source["clientRevision"];
	        this.clientDirty = source["clientDirty"];
	        this.userCount = source["userCount"];
	        this.totalDocuments = source["totalDocuments"];
	        this.dbSize = source["dbSize"];
	        this.generatedAt = this.convertValues(source["generatedAt"], null);
	        this.service = this.convertValues(source["service"], SystemServiceStatistics);
	        this.usage = this.convertValues(source["usage"], SystemUsageStatistics);
	        this.api = this.convertValues(source["api"], SystemAPIStatistics);
	        this.database = this.convertValues(source["database"], SystemDatabaseStatistics);
	        this.outbox = this.convertValues(source["outbox"], SystemOutboxStatistics);
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
	
	export class UpdateProfileRequest {
	    login: string;
	    lastName: string;
	    firstName: string;
	    patronymic: string;
	    noPatronymic: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateProfileRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.login = source["login"];
	        this.lastName = source["lastName"];
	        this.firstName = source["firstName"];
	        this.patronymic = source["patronymic"];
	        this.noPatronymic = source["noPatronymic"];
	    }
	}
	export class UserDocumentPermissionRule {
	    kindCode: string;
	    action: string;
	    isAllowed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UserDocumentPermissionRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kindCode = source["kindCode"];
	        this.action = source["action"];
	        this.isAllowed = source["isAllowed"];
	    }
	}
	export class UserSystemPermissionRule {
	    permission: string;
	    isAllowed: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UserSystemPermissionRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.permission = source["permission"];
	        this.isAllowed = source["isAllowed"];
	    }
	}
	export class UpdateUserDocumentAccessRequest {
	    userId: string;
	    systemPermissions: UserSystemPermissionRule[];
	    permissions: UserDocumentPermissionRule[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateUserDocumentAccessRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userId = source["userId"];
	        this.systemPermissions = this.convertValues(source["systemPermissions"], UserSystemPermissionRule);
	        this.permissions = this.convertValues(source["permissions"], UserDocumentPermissionRule);
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
	export class UpdateUserRequest {
	    id: string;
	    login: string;
	    lastName: string;
	    firstName: string;
	    patronymic: string;
	    noPatronymic: boolean;
	    isActive: boolean;
	    departmentId: string;
	    isDocumentParticipant: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateUserRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.login = source["login"];
	        this.lastName = source["lastName"];
	        this.firstName = source["firstName"];
	        this.patronymic = source["patronymic"];
	        this.noPatronymic = source["noPatronymic"];
	        this.isActive = source["isActive"];
	        this.departmentId = source["departmentId"];
	        this.isDocumentParticipant = source["isDocumentParticipant"];
	    }
	}
	export class UserDocumentAccessProfile {
	    systemPermissions: UserSystemPermissionRule[];
	    permissions: UserDocumentPermissionRule[];
	
	    static createFrom(source: any = {}) {
	        return new UserDocumentAccessProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.systemPermissions = this.convertValues(source["systemPermissions"], UserSystemPermissionRule);
	        this.permissions = this.convertValues(source["permissions"], UserDocumentPermissionRule);
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
	
	export class UserEventFilter {
	    page: number;
	    pageSize: number;
	
	    static createFrom(source: any = {}) {
	        return new UserEventFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.page = source["page"];
	        this.pageSize = source["pageSize"];
	    }
	}

}

export namespace serverclient {
	
	export class SessionState {
	    revision: number;
	    authenticated: boolean;
	    userId: string;
	    reason: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.authenticated = source["authenticated"];
	        this.userId = source["userId"];
	        this.reason = source["reason"];
	    }
	}

}

