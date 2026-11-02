import webapi from "./gocliRequest"
import * as components from "./outletComponents"
export * from "./outletComponents"

/**
 * @description 
 */
export function listMCPAPIKeys() {
	return webapi.get<components.ListMCPAPIKeysResponse>(`/api/admin/api-keys`)
}

/**
 * @description 
 * @param req
 */
export function createMCPAPIKey(req: components.CreateMCPAPIKeyRequest) {
	return webapi.post<components.CreateMCPAPIKeyResponse>(`/api/admin/api-keys`, req)
}

/**
 * @description 
 * @param params
 */
export function revokeMCPAPIKey(params: components.RevokeMCPAPIKeyRequestParams, id: string) {
	return webapi.delete<components.RevokeMCPAPIKeyResponse>(`/api/admin/api-keys/${id}`, params)
}

/**
 * @description "Accept an invitation to a brand as the signed-in person (the invitation's email must be theirs)"
 * @param req
 */
export function acceptInvitation(req: components.AcceptInvitationRequest) {
	return webapi.post<components.Response>(`/api/admin/auth/accept-invitation`, req)
}

/**
 * @description 
 */
export function logout() {
	return webapi.post<components.AnalyticsResponse>(`/api/admin/auth/logout`)
}

/**
 * @description 
 */
export function getCurrentUser() {
	return webapi.get<components.UserInfo>(`/api/admin/auth/me`)
}

/**
 * @description 
 * @param req
 */
export function createBackup(req: components.CreateBackupRequest) {
	return webapi.post<components.CreateBackupResponse>(`/api/admin/backup`, req)
}

/**
 * @description 
 * @param params
 */
export function listBackups(params: components.ListBackupsRequestParams) {
	return webapi.get<components.ListBackupsResponse>(`/api/admin/backup`, params)
}

/**
 * @description 
 * @param params
 */
export function getBackup(params: components.GetBackupRequestParams, id: string) {
	return webapi.get<components.BackupInfo>(`/api/admin/backup/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function deleteBackup(params: components.DeleteBackupRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/backup/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function downloadBackup(params: components.DownloadBackupRequestParams, id: string) {
	return webapi.get<null>(`/api/admin/backup/${id}/download`, params)
}

/**
 * @description 
 */
export function getBackupSettings() {
	return webapi.get<components.BackupSettingsResponse>(`/api/admin/backup/settings`)
}

/**
 * @description 
 * @param req
 */
export function updateBackupSettings(req: components.BackupSettingsRequest) {
	return webapi.put<components.BackupSettingsResponse>(`/api/admin/backup/settings`, req)
}

/**
 * @description 
 * @param params
 */
export function listBlockedDomains(params: components.ListBlockedDomainsRequestParams) {
	return webapi.get<components.ListBlockedDomainsResponse>(`/api/admin/blocked-domains`, params)
}

/**
 * @description 
 * @param req
 */
export function createBlockedDomain(req: components.CreateBlockedDomainRequest) {
	return webapi.post<components.BlockedDomainInfo>(`/api/admin/blocked-domains`, req)
}

/**
 * @description 
 * @param params
 */
export function deleteBlockedDomain(params: components.DeleteBlockedDomainRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/blocked-domains/${id}`, params)
}

/**
 * @description 
 * @param req
 */
export function bulkBlockDomains(req: components.BulkBlockDomainsRequest) {
	return webapi.post<components.BulkBlockDomainsResponse>(`/api/admin/blocked-domains/bulk`, req)
}

/**
 * @description 
 * @param params
 */
export function listSuppressedEmails(params: components.ListSuppressedEmailsRequestParams) {
	return webapi.get<components.ListSuppressedEmailsResponse>(`/api/admin/suppression-list`, params)
}

/**
 * @description 
 * @param req
 */
export function addSuppressedEmail(req: components.AddSuppressedEmailRequest) {
	return webapi.post<components.SuppressedEmailInfo>(`/api/admin/suppression-list`, req)
}

/**
 * @description 
 */
export function clearSuppressionList() {
	return webapi.delete<components.Response>(`/api/admin/suppression-list`)
}

/**
 * @description 
 * @param params
 */
export function deleteSuppressedEmail(params: components.DeleteSuppressedEmailRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/suppression-list/${id}`, params)
}

/**
 * @description 
 * @param req
 */
export function bulkSuppressEmails(req: components.BulkSuppressEmailsRequest) {
	return webapi.post<components.BulkSuppressEmailsResponse>(`/api/admin/suppression-list/bulk`, req)
}

/**
 * @description 
 * @param params
 */
export function listCampaigns(params: components.ListCampaignsRequestParams) {
	return webapi.get<components.ListCampaignsResponse>(`/api/admin/campaigns`, params)
}

/**
 * @description 
 * @param req
 */
export function createCampaign(req: components.CreateCampaignRequest) {
	return webapi.post<components.CampaignInfo>(`/api/admin/campaigns`, req)
}

/**
 * @description 
 * @param params
 */
export function getCampaign(params: components.GetCampaignRequestParams, id: string) {
	return webapi.get<components.CampaignInfo>(`/api/admin/campaigns/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateCampaign(params: components.UpdateCampaignRequestParams, req: components.UpdateCampaignRequest, id: string) {
	return webapi.put<components.CampaignInfo>(`/api/admin/campaigns/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteCampaign(params: components.DeleteCampaignRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/campaigns/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function scheduleCampaign(params: components.ScheduleCampaignRequestParams, req: components.ScheduleCampaignRequest, id: string) {
	return webapi.post<components.CampaignInfo>(`/api/admin/campaigns/${id}/schedule`, params, req)
}

/**
 * @description 
 * @param params
 */
export function sendCampaignNow(params: components.SendCampaignNowRequestParams, id: string) {
	return webapi.post<components.CampaignInfo>(`/api/admin/campaigns/${id}/send`, params)
}

/**
 * @description 
 * @param params
 */
export function getCampaignStats(params: components.GetCampaignRequestParams, id: string) {
	return webapi.get<components.CampaignStatsResponse>(`/api/admin/campaigns/${id}/stats`, params)
}

/**
 * @description 
 * @param params
 */
export function listEmailDesigns(params: components.ListEmailDesignsRequestParams) {
	return webapi.get<components.ListEmailDesignsResponse>(`/api/admin/email-designs`, params)
}

/**
 * @description 
 * @param req
 */
export function createEmailDesign(req: components.CreateEmailDesignRequest) {
	return webapi.post<components.EmailDesignInfo>(`/api/admin/email-designs`, req)
}

/**
 * @description 
 * @param params
 */
export function getEmailDesign(params: components.GetEmailDesignRequestParams, id: string) {
	return webapi.get<components.EmailDesignInfo>(`/api/admin/email-designs/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateEmailDesign(params: components.UpdateEmailDesignRequestParams, req: components.UpdateEmailDesignRequest, id: string) {
	return webapi.put<components.EmailDesignInfo>(`/api/admin/email-designs/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteEmailDesign(params: components.DeleteEmailDesignRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/email-designs/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function listDomainIdentities(params: components.ListDomainIdentitiesRequestParams, org_id: string) {
	return webapi.get<components.ListDomainIdentitiesResponse>(`/api/admin/organizations/${org_id}/domain-identities`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function createDomainIdentity(params: components.CreateDomainIdentityRequestParams, req: components.CreateDomainIdentityRequest, org_id: string) {
	return webapi.post<components.DomainIdentityInfo>(`/api/admin/organizations/${org_id}/domain-identities`, params, req)
}

/**
 * @description 
 * @param params
 */
export function getDomainIdentity(params: components.GetDomainIdentityRequestParams, org_id: string, id: string) {
	return webapi.get<components.DomainIdentityInfo>(`/api/admin/organizations/${org_id}/domain-identities/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function deleteDomainIdentity(params: components.DeleteDomainIdentityRequestParams, org_id: string, id: string) {
	return webapi.delete<components.Response>(`/api/admin/organizations/${org_id}/domain-identities/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function refreshDomainIdentity(params: components.RefreshDomainIdentityRequestParams, org_id: string, id: string) {
	return webapi.post<components.DomainIdentityInfo>(`/api/admin/organizations/${org_id}/domain-identities/${id}/refresh`, params)
}

/**
 * @description 
 * @param params
 */
export function getOrgEmailConfig(params: components.GetOrgEmailConfigRequestParams, org_id: string) {
	return webapi.get<components.OrgEmailConfigInfo>(`/api/admin/organizations/${org_id}/email-config`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateOrgEmailConfig(params: components.UpdateOrgEmailConfigRequestParams, req: components.UpdateOrgEmailConfigRequest, org_id: string) {
	return webapi.put<components.OrgEmailConfigInfo>(`/api/admin/organizations/${org_id}/email-config`, params, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function detectSESQuota(params: components.DetectSESQuotaRequestParams, req: components.DetectSESQuotaRequest, org_id: string) {
	return webapi.post<components.SESQuotaResponse>(`/api/admin/organizations/${org_id}/email-config/detect-quota`, params, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function deleteContactData(params: components.GDPRDeleteRequestParams, req: components.GDPRDeleteRequest, contact_id: string) {
	return webapi.delete<components.GDPRDeleteResponse>(`/api/admin/gdpr/contacts/${contact_id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function getContactConsent(params: components.GDPRConsentRequestParams, contact_id: string) {
	return webapi.get<components.GDPRConsentInfo>(`/api/admin/gdpr/contacts/${contact_id}/consent`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateContactConsent(params: components.UpdateGDPRConsentRequestParams, req: components.UpdateGDPRConsentRequest, contact_id: string) {
	return webapi.put<components.GDPRConsentInfo>(`/api/admin/gdpr/contacts/${contact_id}/consent`, params, req)
}

/**
 * @description 
 * @param params
 */
export function exportContactData(params: components.GDPRExportRequestParams, contact_id: string) {
	return webapi.post<components.GDPRExportResponse>(`/api/admin/gdpr/contacts/${contact_id}/export`, params)
}

/**
 * @description 
 * @param params
 */
export function lookupContact(params: components.GDPRLookupRequestParams) {
	return webapi.get<components.GDPRLookupResponse>(`/api/admin/gdpr/contacts/lookup`, params)
}

/**
 * @description 
 * @param req
 */
export function housekeepingInactive(req: components.HousekeepingInactiveRequest) {
	return webapi.post<components.HousekeepingResponse>(`/api/admin/housekeeping/inactive`, req)
}

/**
 * @description 
 * @param req
 */
export function housekeepingUnconfirmed(req: components.HousekeepingUnconfirmedRequest) {
	return webapi.post<components.HousekeepingResponse>(`/api/admin/housekeeping/unconfirmed`, req)
}

/**
 * @description 
 * @param params
 */
export function listImportJobs(params: components.ListImportJobsRequestParams) {
	return webapi.get<components.ListImportJobsResponse>(`/api/admin/import-jobs`, params)
}

/**
 * @description 
 * @param params
 */
export function getImportJob(params: components.GetImportJobRequestParams, id: string) {
	return webapi.get<components.ImportJobInfo>(`/api/admin/import-jobs/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function cancelImportJob(params: components.CancelImportJobRequestParams, id: string) {
	return webapi.post<components.Response>(`/api/admin/import-jobs/${id}/cancel`, params)
}

/**
 * @description 
 * @param params
 */
export function getEmailDashboardStats(params: components.EmailDashboardStatsRequestParams) {
	return webapi.get<components.EmailDashboardStatsResponse>(`/api/admin/email-stats`, params)
}

/**
 * @description 
 */
export function listLists() {
	return webapi.get<components.ListListsResponse>(`/api/admin/lists`)
}

/**
 * @description 
 * @param req
 */
export function createList(req: components.CreateListRequest) {
	return webapi.post<components.ListInfo>(`/api/admin/lists`, req)
}

/**
 * @description 
 * @param params
 */
export function getList(params: components.GetListRequestParams, id: string) {
	return webapi.get<components.ListInfo>(`/api/admin/lists/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateList(params: components.UpdateListRequestParams, req: components.UpdateListRequest, id: string) {
	return webapi.put<components.ListInfo>(`/api/admin/lists/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteList(params: components.DeleteListRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/lists/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function getListEmbedCode(params: components.GetEmbedCodeRequestParams, id: string) {
	return webapi.get<components.EmbedCodeResponse>(`/api/admin/lists/${id}/embed-code`, params)
}

/**
 * @description 
 * @param params
 */
export function getListStats(params: components.GetListRequestParams, id: string) {
	return webapi.get<components.ListStatsResponse>(`/api/admin/lists/${id}/stats`, params)
}

/**
 * @description 
 * @param params
 */
export function listListSubscribers(params: components.ListSubscribersRequestParams, id: string) {
	return webapi.get<components.ListSubscribersResponse>(`/api/admin/lists/${id}/subscribers`, params)
}

/**
 * @description 
 * @param params
 */
export function removeListSubscriber(params: components.RemoveSubscriberRequestParams, id: string, subscriberId: string) {
	return webapi.delete<components.Response>(`/api/admin/lists/${id}/subscribers/${subscriberId}`, params)
}

/**
 * @description 
 * @param params
 */
export function getSubscriberDetail(params: components.GetSubscriberDetailRequestParams, id: string, subscriberId: string) {
	return webapi.get<components.SubscriberDetailResponse>(`/api/admin/lists/${id}/subscribers/${subscriberId}`, params)
}

/**
 * @description 
 * @param params
 */
export function listCustomFields(params: components.ListCustomFieldsRequestParams, listId: string) {
	return webapi.get<components.ListCustomFieldsResponse>(`/api/admin/lists/${listId}/custom-fields`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function createCustomField(params: components.CreateCustomFieldRequestParams, req: components.CreateCustomFieldRequest, listId: string) {
	return webapi.post<components.CustomFieldInfo>(`/api/admin/lists/${listId}/custom-fields`, params, req)
}

/**
 * @description 
 * @param params
 */
export function getCustomField(params: components.GetCustomFieldRequestParams, listId: string, fieldId: string) {
	return webapi.get<components.CustomFieldInfo>(`/api/admin/lists/${listId}/custom-fields/${fieldId}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateCustomField(params: components.UpdateCustomFieldRequestParams, req: components.UpdateCustomFieldRequest, listId: string, fieldId: string) {
	return webapi.put<components.CustomFieldInfo>(`/api/admin/lists/${listId}/custom-fields/${fieldId}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteCustomField(params: components.DeleteCustomFieldRequestParams, listId: string, fieldId: string) {
	return webapi.delete<components.Response>(`/api/admin/lists/${listId}/custom-fields/${fieldId}`, params)
}

/**
 * @description 
 */
export function listOrganizations() {
	return webapi.get<components.OrgListResponse>(`/api/admin/organizations`)
}

/**
 * @description 
 * @param req
 */
export function createOrganization(req: components.CreateOrgRequest) {
	return webapi.post<components.OrgInfo>(`/api/admin/organizations`, req)
}

/**
 * @description 
 */
export function getAllOrganizationsStats() {
	return webapi.get<components.OrgBulkStatsResponse>(`/api/admin/organizations-stats`)
}

/**
 * @description 
 * @param params
 */
export function getOrganization(params: components.GetOrgRequestParams, id: string) {
	return webapi.get<components.OrgInfo>(`/api/admin/organizations/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateOrganization(params: components.UpdateOrgRequestParams, req: components.UpdateOrgRequest, id: string) {
	return webapi.put<components.OrgInfo>(`/api/admin/organizations/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteOrganization(params: components.DeleteOrgRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/organizations/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function getDashboardStats(params: components.GetDashboardStatsRequestParams, id: string) {
	return webapi.get<components.DashboardStatsResponse>(`/api/admin/organizations/${id}/dashboard-stats`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateOrgEmailSettings(params: components.UpdateOrgEmailSettingsRequestParams, req: components.UpdateOrgEmailSettingsRequest, id: string) {
	return webapi.put<components.OrgInfo>(`/api/admin/organizations/${id}/email`, params, req)
}

/**
 * @description "Invite a person to a brand. Outlet is invite-only: the invitation decides the brand and role."
 * @param params
 * @param req
 */
export function createInvitation(params: components.CreateInvitationRequestParams, req: components.CreateInvitationRequest, id: string) {
	return webapi.post<components.CreateInvitationResponse>(`/api/admin/organizations/${id}/invitations`, params, req)
}

/**
 * @description 
 * @param params
 */
export function regenerateOrgApiKey(params: components.RegenerateApiKeyRequestParams, id: string) {
	return webapi.post<components.OrgInfo>(`/api/admin/organizations/${id}/regenerate-key`, params)
}

/**
 * @description 
 * @param params
 */
export function getOrganizationBySlug(params: components.GetOrgBySlugRequestParams, slug: string) {
	return webapi.get<components.OrgInfo>(`/api/admin/organizations/slug/${slug}`, params)
}

/**
 * @description 
 */
export function listPoolDomains() {
	return webapi.get<components.ListPoolDomainsResponse>(`/api/admin/pool/domains`)
}

/**
 * @description 
 * @param req
 */
export function createPoolDomain(req: components.CreatePoolDomainRequest) {
	return webapi.post<components.PoolDomainInfo>(`/api/admin/pool/domains`, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function setPoolDomainStatus(params: components.SetPoolStatusRequestParams, req: components.SetPoolStatusRequest, id: string) {
	return webapi.put<components.PoolDomainInfo>(`/api/admin/pool/domains/${id}/status`, params, req)
}

/**
 * @description 
 */
export function listPoolMailboxes() {
	return webapi.get<components.ListPoolMailboxesResponse>(`/api/admin/pool/mailboxes`)
}

/**
 * @description 
 * @param req
 */
export function createPoolMailbox(req: components.CreatePoolMailboxRequest) {
	return webapi.post<components.PoolMailboxInfo>(`/api/admin/pool/mailboxes`, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function setPoolMailboxStatus(params: components.SetPoolStatusRequestParams, req: components.SetPoolStatusRequest, id: string) {
	return webapi.put<components.PoolMailboxInfo>(`/api/admin/pool/mailboxes/${id}/status`, params, req)
}

/**
 * @description 
 * @param req
 */
export function clearPoolTenantHold(req: components.ClearPoolTenantHoldRequest) {
	return webapi.post<components.ClearPoolTenantHoldResponse>(`/api/admin/pool/tenants/clear-hold`, req)
}

/**
 * @description 
 * @param params
 */
export function listEmailQueue(params: components.EmailQueueListRequestParams) {
	return webapi.get<components.EmailQueueListResponse>(`/api/admin/email-queue`, params)
}

/**
 * @description 
 * @param params
 */
export function cancelEmail(params: components.CancelEmailRequestParams, id: string) {
	return webapi.post<components.Response>(`/api/admin/email-queue/${id}/cancel`, params)
}

/**
 * @description 
 * @param req
 */
export function createEntryRule(req: components.CreateEntryRuleRequest) {
	return webapi.post<components.EntryRuleResponse>(`/api/admin/entry-rules`, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateEntryRule(params: components.UpdateEntryRuleRequestParams, req: components.UpdateEntryRuleRequest, id: string) {
	return webapi.put<components.EntryRuleResponse>(`/api/admin/entry-rules/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteEntryRule(params: components.DeleteEntryRuleRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/entry-rules/${id}`, params)
}

/**
 * @description 
 */
export function listSequences() {
	return webapi.get<components.SequenceListResponse>(`/api/admin/sequences`)
}

/**
 * @description 
 * @param req
 */
export function createSequence(req: components.CreateSequenceRequest) {
	return webapi.post<components.SequenceInfo>(`/api/admin/sequences`, req)
}

/**
 * @description 
 * @param params
 */
export function getSequence(params: components.GetSequenceRequestParams, id: string) {
	return webapi.get<components.SequenceDetailResponse>(`/api/admin/sequences/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateSequence(params: components.UpdateSequenceRequestParams, req: components.UpdateSequenceRequest, id: string) {
	return webapi.put<components.SequenceInfo>(`/api/admin/sequences/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteSequence(params: components.GetSequenceRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/sequences/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function getSequenceStats(params: components.SequenceStatsRequestParams, id: string) {
	return webapi.get<components.SequenceStatsResponse>(`/api/admin/sequences/${id}/stats`, params)
}

/**
 * @description 
 * @param req
 */
export function createTemplate(req: components.CreateTemplateRequest) {
	return webapi.post<components.TemplateInfo>(`/api/admin/templates`, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateTemplate(params: components.UpdateTemplateRequestParams, req: components.UpdateTemplateRequest, id: string) {
	return webapi.put<components.TemplateInfo>(`/api/admin/templates/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteTemplate(params: components.DeleteTemplateRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/templates/${id}`, params)
}

/**
 * @description 
 */
export function getPlatformSettings() {
	return webapi.get<components.GetPlatformSettingsResponse>(`/api/admin/settings`)
}

/**
 * @description 
 * @param params
 */
export function getPlatformSettingsByCategory(params: components.GetPlatformSettingsByCategoryRequestParams, category: string) {
	return webapi.get<components.GetPlatformSettingsResponse>(`/api/admin/settings/${category}`, params)
}

/**
 * @description 
 * @param req
 */
export function updateEmailSettings(req: components.UpdateEmailSettingsRequest) {
	return webapi.put<components.UpdateSettingsResponse>(`/api/admin/settings/email`, req)
}

/**
 * @description 
 */
export function getPlatformSESQuota() {
	return webapi.get<components.SESQuotaResponse>(`/api/admin/settings/ses-quota`)
}

/**
 * @description 
 */
export function getSetupStatus() {
	return webapi.get<components.SetupStatusResponse>(`/api/admin/settings/setup-status`)
}

/**
 * @description 
 * @param params
 */
export function blockSubscriber(params: components.SubscriberActionRequestParams, id: string) {
	return webapi.post<components.SubscriberInfo>(`/api/admin/subscribers/${id}/block`, params)
}

/**
 * @description 
 * @param params
 */
export function resubscribeSubscriber(params: components.SubscriberActionRequestParams, id: string) {
	return webapi.post<components.SubscriberInfo>(`/api/admin/subscribers/${id}/resubscribe`, params)
}

/**
 * @description 
 * @param params
 */
export function unblockSubscriber(params: components.SubscriberActionRequestParams, id: string) {
	return webapi.post<components.SubscriberInfo>(`/api/admin/subscribers/${id}/unblock`, params)
}

/**
 * @description 
 * @param params
 */
export function unsubscribeSubscriber(params: components.SubscriberActionRequestParams, id: string) {
	return webapi.post<components.SubscriberInfo>(`/api/admin/subscribers/${id}/unsubscribe`, params)
}

/**
 * @description 
 * @param params
 */
export function verifySubscriber(params: components.SubscriberActionRequestParams, id: string) {
	return webapi.post<components.SubscriberInfo>(`/api/admin/subscribers/${id}/verify`, params)
}

/**
 * @description 
 */
export function checkForUpdates() {
	return webapi.get<components.UpdateCheckResponse>(`/api/admin/system/updates/check`)
}

/**
 * @description 
 */
export function getVersion() {
	return webapi.get<components.VersionInfo>(`/api/admin/system/version`)
}

/**
 * @description 
 */
export function listTransactionalEmails() {
	return webapi.get<components.ListTransactionalEmailsResponse>(`/api/admin/transactional-emails`)
}

/**
 * @description 
 * @param req
 */
export function createTransactionalEmail(req: components.CreateTransactionalEmailRequest) {
	return webapi.post<components.TransactionalEmailInfo>(`/api/admin/transactional-emails`, req)
}

/**
 * @description 
 * @param params
 */
export function getTransactionalEmail(params: components.GetTransactionalEmailRequestParams, id: string) {
	return webapi.get<components.TransactionalEmailInfo>(`/api/admin/transactional-emails/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateTransactionalEmail(params: components.UpdateTransactionalEmailRequestParams, req: components.UpdateTransactionalEmailRequest, id: string) {
	return webapi.put<components.TransactionalEmailInfo>(`/api/admin/transactional-emails/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteTransactionalEmail(params: components.DeleteTransactionalEmailRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/transactional-emails/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function getTransactionalEmailStats(params: components.GetTransactionalEmailRequestParams, id: string) {
	return webapi.get<components.TransactionalStatsResponse>(`/api/admin/transactional-emails/${id}/stats`, params)
}

/**
 * @description 
 */
export function listUsers() {
	return webapi.get<components.UserListResponse>(`/api/admin/users`)
}

/**
 * @description 
 * @param req
 */
export function createUser(req: components.CreateUserRequest) {
	return webapi.post<components.CreateUserResponse>(`/api/admin/users`, req)
}

/**
 * @description 
 * @param params
 */
export function getUser(params: components.GetUserRequestParams, id: string) {
	return webapi.get<components.UserInfo>(`/api/admin/users/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateUser(params: components.UpdateUserRequestParams, req: components.UpdateUserRequest, id: string) {
	return webapi.put<components.UserInfo>(`/api/admin/users/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteUser(params: components.DeleteUserRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/users/${id}`, params)
}

/**
 * @description 
 */
export function adminListWebhooks() {
	return webapi.get<components.ListWebhooksResponse>(`/api/admin/webhooks`)
}

/**
 * @description 
 * @param req
 */
export function adminCreateWebhook(req: components.RegisterWebhookRequest) {
	return webapi.post<components.RegisterWebhookResponse>(`/api/admin/webhooks`, req)
}

/**
 * @description 
 * @param params
 */
export function adminGetWebhook(params: components.GetWebhookRequestParams, id: string) {
	return webapi.get<components.WebhookInfo>(`/api/admin/webhooks/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function adminUpdateWebhook(params: components.UpdateWebhookRequestParams, req: components.UpdateWebhookRequest, id: string) {
	return webapi.put<components.WebhookInfo>(`/api/admin/webhooks/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function adminDeleteWebhook(params: components.DeleteWebhookRequestParams, id: string) {
	return webapi.delete<components.Response>(`/api/admin/webhooks/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function adminListWebhookLogs(params: components.ListWebhookLogsRequestParams, id: string) {
	return webapi.get<components.ListWebhookLogsResponse>(`/api/admin/webhooks/${id}/logs`, params)
}

/**
 * @description 
 * @param params
 */
export function adminTestWebhook(params: components.TestWebhookRequestParams, id: string) {
	return webapi.post<components.TestWebhookResponse>(`/api/admin/webhooks/${id}/test`, params)
}

/**
 * @description "Request password reset"
 * @param req
 */
export function forgotPassword(req: components.ForgotPasswordRequest) {
	return webapi.post<components.ForgotPasswordResponse>(`/api/v1/auth/forgot-password`, req)
}

/**
 * @description "User login"
 * @param req
 */
export function login(req: components.LoginRequest) {
	return webapi.post<components.LoginResponse>(`/api/v1/auth/login`, req)
}

/**
 * @description "Refresh access token using refresh token"
 * @param req
 */
export function refreshToken(req: components.RefreshTokenRequest) {
	return webapi.post<components.LoginResponse>(`/api/v1/auth/refresh`, req)
}

/**
 * @description "Create an account from an invitation (Outlet is invite-only)"
 * @param req
 */
export function register(req: components.RegisterRequest) {
	return webapi.post<components.RegisterResponse>(`/api/v1/auth/register`, req)
}

/**
 * @description "Reset password with token"
 * @param req
 */
export function resetPassword(req: components.ResetPasswordRequest) {
	return webapi.post<components.ResetPasswordResponse>(`/api/v1/auth/reset-password`, req)
}

/**
 * @description "Validate invitation token and get user details"
 * @param params
 */
export function validateInvitation(params: components.ValidateInvitationRequestParams) {
	return webapi.get<components.ValidateInvitationResponse>(`/api/v1/auth/validate-invitation`, params)
}

/**
 * @description "Verify email address"
 * @param params
 */
export function verifyEmail(params: components.VerifyEmailRequestParams) {
	return webapi.get<components.VerifyEmailResponse>(`/api/v1/auth/verify-email`, params)
}

/**
 * @description 
 */
export function health() {
	return webapi.get<components.HealthResponse>(`/api/v1/health`)
}

/**
 * @description "Create initial admin account (only works when no admin exists)"
 * @param req
 */
export function createInitialAdmin(req: components.CreateInitialAdminRequest) {
	return webapi.post<components.CreateInitialAdminResponse>(`/api/v1/setup/admin`, req)
}

/**
 * @description "Check if initial setup is required"
 */
export function getPublicSetupStatus() {
	return webapi.get<components.SetupStatusResponse>(`/api/v1/setup/status`)
}

/**
 * @description 
 * @param req
 */
export function createContact(req: components.ContactRequest) {
	return webapi.post<components.ContactResponse>(`/sdk/v1/contacts`, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function subscribeToList(params: components.SubscribeRequestParams, req: components.SubscribeRequest, slug: string) {
	return webapi.post<components.Response>(`/sdk/v1/lists/${slug}/subscribe`, params, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function unsubscribeFromList(params: components.SubscribeRequestParams, req: components.SubscribeRequest, slug: string) {
	return webapi.post<components.Response>(`/sdk/v1/lists/${slug}/unsubscribe`, params, req)
}

/**
 * @description 
 * @param params
 */
export function getContact(params: components.GetContactRequestParams, id: string) {
	return webapi.get<components.SDKContactInfo>(`/sdk/v1/contacts/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateContact(params: components.UpdateContactRequestParams, req: components.UpdateContactRequest, id: string) {
	return webapi.put<components.SDKContactInfo>(`/sdk/v1/contacts/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function listContactActivity(params: components.ListContactActivityRequestParams, id: string) {
	return webapi.get<components.ListContactActivityResponse>(`/sdk/v1/contacts/${id}/activity`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function addContactTags(params: components.AddContactTagsRequestParams, req: components.AddContactTagsRequest, id: string) {
	return webapi.post<components.Response>(`/sdk/v1/contacts/${id}/tags`, params, req)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function removeContactTags(params: components.RemoveContactTagsRequestParams, req: components.RemoveContactTagsRequest, id: string) {
	return webapi.delete<components.Response>(`/sdk/v1/contacts/${id}/tags`, params, req)
}

/**
 * @description 
 * @param req
 */
export function globalUnsubscribe(req: components.GlobalUnsubscribeRequest) {
	return webapi.post<components.Response>(`/sdk/v1/contacts/unsubscribe`, req)
}

/**
 * @description 
 * @param req
 */
export function sendEmail(req: components.SendEmailRequest) {
	return webapi.post<components.SendEmailResponse>(`/sdk/v1/emails`, req)
}

/**
 * @description 
 * @param params
 */
export function getEmailStatus(params: components.GetEmailStatusRequestParams, messageId: string) {
	return webapi.get<components.EmailStatusResponse>(`/sdk/v1/emails/${messageId}`, params)
}

/**
 * @description 
 * @param params
 */
export function listEmailEvents(params: components.ListEmailEventsRequestParams, messageId: string) {
	return webapi.get<components.ListEmailEventsResponse>(`/sdk/v1/emails/${messageId}/events`, params)
}

/**
 * @description "Assign the tenant free mailboxes"
 * @param req
 */
export function poolAssign(req: components.PoolAssignRequest) {
	return webapi.post<components.PoolAssignResponse>(`/sdk/v1/pool/assign`, req)
}

/**
 * @description "Take a bounce (RFC 3464 DSN) or complaint (RFC 5965 ARF) report"
 * @param req
 */
export function poolReport(req: components.PoolReportRequest) {
	return webapi.post<components.PoolReportResponse>(`/sdk/v1/pool/bounces`, req)
}

/**
 * @description "The tenant's mailboxes with today's allowance left"
 * @param params
 */
export function poolCapacity(params: components.PoolCapacityRequestParams) {
	return webapi.get<components.PoolCapacityResponse>(`/sdk/v1/pool/capacity`, params)
}

/**
 * @description "Release every mailbox of the tenant"
 * @param req
 */
export function poolRelease(req: components.PoolReleaseRequest) {
	return webapi.post<components.PoolReleaseResponse>(`/sdk/v1/pool/release`, req)
}

/**
 * @description "Send one email from the tenant's sticky pool mailbox"
 * @param req
 */
export function poolSend(req: components.PoolSendRequest) {
	return webapi.post<components.PoolSendResponse>(`/sdk/v1/pool/send`, req)
}

/**
 * @description "Check addresses before their first email (no SMTP probing)"
 * @param req
 */
export function poolVerify(req: components.PoolVerifyRequest) {
	return webapi.post<components.PoolVerifyResponse>(`/sdk/v1/pool/verify`, req)
}

/**
 * @description 
 * @param req
 */
export function enrollInSequence(req: components.EnrollSequenceRequest) {
	return webapi.post<components.EnrollSequenceResponse>(`/sdk/v1/sequences/enroll`, req)
}

/**
 * @description 
 * @param params
 */
export function getSequenceEnrollments(params: components.GetSequenceEnrollmentRequestParams) {
	return webapi.get<components.ListSequenceEnrollmentsResponse>(`/sdk/v1/sequences/enrollments`, params)
}

/**
 * @description 
 * @param req
 */
export function pauseSequenceEnrollment(req: components.PauseSequenceRequest) {
	return webapi.post<components.Response>(`/sdk/v1/sequences/pause`, req)
}

/**
 * @description 
 * @param req
 */
export function resumeSequenceEnrollment(req: components.ResumeSequenceRequest) {
	return webapi.post<components.Response>(`/sdk/v1/sequences/resume`, req)
}

/**
 * @description 
 * @param req
 */
export function unenrollFromSequence(req: components.UnenrollSequenceRequest) {
	return webapi.post<components.Response>(`/sdk/v1/sequences/unenroll`, req)
}

/**
 * @description 
 * @param params
 */
export function getContactStats(params: components.GetContactStatsRequestParams) {
	return webapi.get<components.GetContactStatsResponse>(`/sdk/v1/stats/contacts`, params)
}

/**
 * @description 
 * @param params
 */
export function getEmailStats(params: components.GetEmailStatsRequestParams) {
	return webapi.get<components.GetEmailStatsResponse>(`/sdk/v1/stats/emails`, params)
}

/**
 * @description 
 * @param params
 */
export function getStatsOverview(params: components.GetStatsOverviewRequestParams) {
	return webapi.get<components.StatsOverviewResponse>(`/sdk/v1/stats/overview`, params)
}

/**
 * @description 
 * @param req
 */
export function trackClick(req: components.TrackClickRequest) {
	return webapi.post<components.Response>(`/sdk/v1/tracking/click`, req)
}

/**
 * @description 
 * @param req
 */
export function trackConfirm(req: components.TrackConfirmRequest) {
	return webapi.post<components.TrackConfirmResponse>(`/sdk/v1/tracking/confirm`, req)
}

/**
 * @description 
 * @param req
 */
export function trackOpen(req: components.TrackOpenRequest) {
	return webapi.post<components.Response>(`/sdk/v1/tracking/open`, req)
}

/**
 * @description 
 * @param req
 */
export function trackUnsubscribe(req: components.TrackUnsubscribeRequest) {
	return webapi.post<components.Response>(`/sdk/v1/tracking/unsubscribe`, req)
}

/**
 * @description 
 * @param req
 */
export function registerWebhook(req: components.RegisterWebhookRequest) {
	return webapi.post<components.RegisterWebhookResponse>(`/sdk/v1/webhooks`, req)
}

/**
 * @description 
 */
export function listWebhooks() {
	return webapi.get<components.ListWebhooksResponse>(`/sdk/v1/webhooks`)
}

/**
 * @description 
 * @param params
 */
export function getWebhook(params: components.GetWebhookRequestParams, id: string) {
	return webapi.get<components.WebhookInfo>(`/sdk/v1/webhooks/${id}`, params)
}

/**
 * @description 
 * @param params
 * @param req
 */
export function updateWebhook(params: components.UpdateWebhookRequestParams, req: components.UpdateWebhookRequest, id: string) {
	return webapi.put<components.WebhookInfo>(`/sdk/v1/webhooks/${id}`, params, req)
}

/**
 * @description 
 * @param params
 */
export function deleteWebhook(params: components.DeleteWebhookRequestParams, id: string) {
	return webapi.delete<components.Response>(`/sdk/v1/webhooks/${id}`, params)
}

/**
 * @description 
 * @param params
 */
export function listWebhookLogs(params: components.ListWebhookLogsRequestParams, id: string) {
	return webapi.get<components.ListWebhookLogsResponse>(`/sdk/v1/webhooks/${id}/logs`, params)
}

/**
 * @description 
 * @param params
 */
export function testWebhook(params: components.TestWebhookRequestParams, id: string) {
	return webapi.post<components.TestWebhookResponse>(`/sdk/v1/webhooks/${id}/test`, params)
}
