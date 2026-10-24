// Code generated; DO NOT EDIT.
package eventsv2

type eventRule struct {
	Definition, Producer, AggregateType, Topic string
	Encrypted                                  bool
	AggregateField                             string
}

var eventRules = map[string]eventRule{
	"identity.user_registered":         {"UserRegistered", "identity", "user", "identity.events.v2", false, "user_id"},
	"identity.user_logged_in":          {"UserLoggedIn", "identity", "user", "identity.events.v2", false, "user_id"},
	"identity.email_verified":          {"EmailVerified", "identity", "user", "identity.events.v2", false, "user_id"},
	"identity.password_changed":        {"PasswordChanged", "identity", "user", "identity.events.v2", false, "user_id"},
	"identity.mfa_changed":             {"MFAChanged", "identity", "user", "identity.events.v2", false, "user_id"},
	"identity.session_revoked":         {"SessionRevoked", "identity", "user", "identity.events.v2", false, "user_id"},
	"identity.user_banned":             {"UserBanned", "identity", "user", "identity.events.v2", false, "user_id"},
	"identity.user_unbanned":           {"UserUnbanned", "identity", "user", "identity.events.v2", false, "user_id"},
	"notification.email_verification":  {"EmailVerification", "identity", "delivery_command", "notification.commands.v2", true, ""},
	"notification.password_reset":      {"PasswordReset", "identity", "delivery_command", "notification.commands.v2", true, ""},
	"catalog.revision_published":       {"RevisionPublished", "catalog", "plan", "catalog.events.v1", false, "plan_id"},
	"catalog.revision_archived":        {"RevisionArchived", "catalog", "plan", "catalog.events.v1", false, "plan_id"},
	"billing.payment_accepted":         {"PaymentAccepted", "billing", "order", "billing.events.v1", false, "order_id"},
	"billing.order_fulfilled":          {"OrderFulfilled", "billing", "order", "billing.events.v1", false, "order_id"},
	"subscription.action_applied":      {"ActionApplied", "subscription", "subscription_action", "subscription.events.v1", false, "action_id"},
	"subscription.action_rejected":     {"ActionRejected", "subscription", "subscription_action", "subscription.events.v1", false, "action_id"},
	"subscription.changed":             {"SubscriptionChanged", "subscription", "subscription", "subscription.events.v1", false, "subscription_id"},
	"subscription.expired":             {"SubscriptionExpired", "subscription", "subscription", "subscription.events.v1", false, "subscription_id"},
	"subscription.closed":              {"SubscriptionClosed", "subscription", "subscription", "subscription.events.v1", false, "subscription_id"},
	"subscription.compensated":         {"SubscriptionCompensated", "subscription", "subscription", "subscription.events.v1", false, "subscription_id"},
	"referral.account_created":         {"ReferralAccountCreated", "referral", "referral_account", "referral.events.v1", false, "user_id"},
	"referral.attribution_finalized":   {"AttributionFinalized", "referral", "referral_account", "referral.events.v1", false, "user_id"},
	"referral.rate_changed":            {"RateChanged", "referral", "referral_rate", "referral.events.v1", false, ""},
	"referral.commission_posted":       {"CommissionPosted", "referral", "referral_account", "referral.events.v1", false, "user_id"},
	"referral.commission_reversed":     {"CommissionReversed", "referral", "referral_account", "referral.events.v1", false, "user_id"},
	"referral.payout_requested":        {"PayoutRequested", "referral", "referral_payout", "referral.events.v1", false, "payout_id"},
	"referral.payout_decided":          {"PayoutDecided", "referral", "referral_payout", "referral.events.v1", false, "payout_id"},
	"referral.payout_cancelled":        {"PayoutCancelled", "referral", "referral_payout", "referral.events.v1", false, "payout_id"},
	"profile.updated":                  {"ProfileUpdated", "profile", "user", "profile.events.v2", false, "user_id"},
	"profile.telegram_binding_changed": {"TelegramBindingChanged", "profile", "user", "profile.events.v2", false, "user_id"},
	"external.readiness_changed":       {"ExternalReadinessChanged", "profile", "external_event", "external.events.v1", false, ""},
	"external.trading_alert":           {"ExternalTradingAlert", "profile", "external_event", "external.events.v1", false, ""},
	"notification.delivery_replayed":   {"DeliveryReplayed", "notification", "delivery", "notification.events.v2", false, "delivery_id"},
	"notification.template_published":  {"TemplatePublished", "notification", "notification_template", "notification.events.v2", false, ""},
	"referral.claim_prepared":          {"ClaimPrepared", "referral", "referral_claim", "referral.events.v1", false, "claim_id"},
	"referral.quote_created":           {"QuoteCreated", "referral", "referral_quote", "referral.events.v1", false, "quote_id"},
	"identity.admin_action_recorded":   {"AdminActionRecorded", "identity", "identity_admin_action", "identity.events.v2", false, "audit_id"},
	"billing.payment_reviewed":         {"PaymentReviewed", "billing", "billing_review", "billing.events.v1", false, "decision_id"},
}
