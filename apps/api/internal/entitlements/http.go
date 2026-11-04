package entitlements

import (
	stderrors "errors"
	"net/http"

	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/httpjson"
)

// PaymentRequiredCode is the error code of a plan refusal, the one tronc's
// errors.PaymentRequired uses.
const PaymentRequiredCode = "payment_required"

// WriteError answers a plan refusal with HTTP 402 in the suite error
// envelope, plus the feature key, the plan's limit in GB and the Registre page
// where an owner upgrades. Every other error goes to httpjson.WriteError.
func WriteError(w http.ResponseWriter, err error) {
	var refusal *forfait.LimitError
	if !stderrors.As(err, &refusal) {
		httpjson.WriteError(w, err)
		return
	}
	body := map[string]any{
		"code":    PaymentRequiredCode,
		"message": "storage plan limit reached: upgrade the plan or free some space",
		"key":     refusal.Key,
		"limit":   refusal.Limit,
	}
	if refusal.UpgradeURL != "" {
		body["upgrade_url"] = refusal.UpgradeURL
	}
	httpjson.WriteJSON(w, http.StatusPaymentRequired, map[string]any{"error": body})
}
