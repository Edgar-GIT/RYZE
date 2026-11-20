package email

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"strings"
	texttemplate "text/template"

	"ryze/backend/models"
)

// Template identifiers.
const (
	TemplateAccountVerification = "account_verification"
	TemplatePasswordReset       = "password_reset"
	TemplatePasswordChanged     = "password_changed"
	TemplateAccessGranted       = "access_granted"
	TemplateIntakeReceived      = "pl2_intake_received"
	TemplateTrainerAssigned     = "pl2_trainer_assigned"
	TemplateRequestAssigned     = "pl2_request_assigned"
	TemplateServiceDelivered    = "pl2_delivered"
	TemplateSupportReply        = "support_reply"
	TemplateInvoiceIssued       = "invoice_issued"
)

// Supported locales. Anything else falls back to English.
const (
	LocaleEnglish    = "en"
	LocalePortuguese = "pt"
)

// GreetingName is the name an email may greet the account with. Anyone can
// register any address with a name of their choosing, so the name is used
// only once the address owner confirmed it; otherwise emails greet without a
// name and carry no text written by someone else.
func GreetingName(user *models.User) string {
	if user == nil || user.EmailVerifiedAt == nil {
		return ""
	}
	return user.FirstName
}

// NormalizeLocale maps a stored preference to a supported locale.
func NormalizeLocale(locale string) string {
	if strings.EqualFold(strings.TrimSpace(locale), LocalePortuguese) {
		return LocalePortuguese
	}
	return LocaleEnglish
}

// copyText is the wording of one template in one language. Every value is
// plain text: the HTML layout escapes it.
type copyText struct {
	Subject   string
	Heading   string
	Body      string
	Action    string
	Footnote  string
	NeedsLink bool
}

// copies holds the reviewed wording of every template. Placeholders use
// text/template syntax over the template data ({{.Name}}, {{.Program}}).
var copies = map[string]map[string]copyText{
	TemplateAccountVerification: {
		LocaleEnglish: {
			Subject: "Confirm your RYZE email", Heading: "Confirm your email",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, confirm that this is your email address so we can reach you about your account and purchases.",
			Action: "Confirm email", Footnote: "The link expires in 48 hours. If you did not create a RYZE account, ignore this email.",
			NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "Confirma o teu email da RYZE", Heading: "Confirma o teu email",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, confirma que este é o teu endereço de email para te podermos contactar sobre a tua conta e as tuas compras.",
			Action: "Confirmar email", Footnote: "O link expira em 48 horas. Se não criaste uma conta RYZE, ignora este email.",
			NeedsLink: true,
		},
	},
	TemplatePasswordReset: {
		LocaleEnglish: {
			Subject: "Reset your RYZE password", Heading: "Reset your password",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, we received a request to reset the password of your RYZE account.",
			Action: "Choose a new password", Footnote: "The link expires in 60 minutes and works once. If you did not ask for it, ignore this email: your password stays the same.",
			NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "Redefine a tua password da RYZE", Heading: "Redefine a tua password",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, recebemos um pedido para redefinir a password da tua conta RYZE.",
			Action: "Escolher nova password", Footnote: "O link expira em 60 minutos e só funciona uma vez. Se não o pediste, ignora este email: a tua password não muda.",
			NeedsLink: true,
		},
	},
	TemplatePasswordChanged: {
		LocaleEnglish: {
			Subject: "Your RYZE password was changed", Heading: "Password changed",
			Body:     "Hi{{if .Name}} {{.Name}}{{end}}, the password of your RYZE account was just changed and every other session was signed out.",
			Footnote: "If this was not you, reset your password right away and contact RYZE.",
		},
		LocalePortuguese: {
			Subject: "A tua password da RYZE foi alterada", Heading: "Password alterada",
			Body:     "Olá{{if .Name}} {{.Name}}{{end}}, a password da tua conta RYZE acabou de ser alterada e todas as outras sessões foram terminadas.",
			Footnote: "Se não foste tu, redefine já a tua password e contacta a RYZE.",
		},
	},
	TemplateAccessGranted: {
		LocaleEnglish: {
			Subject: "Your RYZE access is ready: {{.Program}}", Heading: "Your access is ready",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, your purchase of {{.Program}} is confirmed and it is now in My Programs.",
			Action: "Open My Programs", Footnote: "You can find every purchase in your account under Purchases.",
			NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "O teu acesso RYZE está pronto: {{.Program}}", Heading: "O teu acesso está pronto",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, a tua compra de {{.Program}} está confirmada e já está em Os meus programas.",
			Action: "Abrir os meus programas", Footnote: "Encontras todas as compras na tua conta, em Compras.",
			NeedsLink: true,
		},
	},
	TemplateIntakeReceived: {
		LocaleEnglish: {
			Subject: "We received your intake: {{.Program}}", Heading: "Intake received",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, your answers for {{.Program}} were submitted. RYZE will now assign your trainer.",
			Action: "Follow your request", NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "Recebemos o teu questionário: {{.Program}}", Heading: "Questionário recebido",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, as tuas respostas para {{.Program}} foram submetidas. A RYZE vai agora atribuir o teu treinador.",
			Action: "Acompanhar o pedido", NeedsLink: true,
		},
	},
	TemplateTrainerAssigned: {
		LocaleEnglish: {
			Subject: "Your trainer is preparing {{.Program}}", Heading: "Your trainer is assigned",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, a RYZE trainer was assigned to your {{.Program}} request and is preparing your programme and nutrition plan.",
			Action: "Follow your request", NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "O teu treinador está a preparar {{.Program}}", Heading: "O teu treinador foi atribuído",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, um treinador RYZE foi atribuído ao teu pedido {{.Program}} e está a preparar o teu programa e o teu plano de nutrição.",
			Action: "Acompanhar o pedido", NeedsLink: true,
		},
	},
	TemplateRequestAssigned: {
		LocaleEnglish: {
			Subject: "New Premium Level 2 request assigned to you", Heading: "New request assigned",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, RYZE assigned a Premium Level 2 request to you. Open it to read the intake and prepare the programme and nutrition plan.",
			Action: "Open the request", NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "Novo pedido Premium Level 2 atribuído a ti", Heading: "Novo pedido atribuído",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, a RYZE atribuiu-te um pedido Premium Level 2. Abre-o para leres o questionário e preparares o programa e o plano de nutrição.",
			Action: "Abrir o pedido", NeedsLink: true,
		},
	},
	TemplateServiceDelivered: {
		LocaleEnglish: {
			Subject: "Your {{.Program}} is ready", Heading: "Your programme is ready",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, your trainer delivered your {{.Program}} programme and nutrition plan. They are now available in your account.",
			Action: "Open your programme", NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "O teu {{.Program}} está pronto", Heading: "O teu programa está pronto",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, o teu treinador entregou o programa e o plano de nutrição {{.Program}}. Já estão disponíveis na tua conta.",
			Action: "Abrir o programa", NeedsLink: true,
		},
	},
	TemplateInvoiceIssued: {
		LocaleEnglish: {
			Subject: "Your RYZE purchase receipt {{.Invoice.Number}}", Heading: "Purchase receipt",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, thank you for your purchase of {{.Program}}. Your receipt is below and is always available in your account under Purchases.",
			Action: "View the receipt", Footnote: "You can print it or save it as a PDF from the receipt page.",
			NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "O teu comprovativo de compra RYZE {{.Invoice.Number}}", Heading: "Comprovativo de compra",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, obrigado pela tua compra de {{.Program}}. O comprovativo está abaixo e fica sempre disponível na tua conta, em Compras.",
			Action: "Ver o comprovativo", Footnote: "Na página do comprovativo podes imprimi-lo ou guardá-lo em PDF.",
			NeedsLink: true,
		},
	},
	TemplateSupportReply: {
		LocaleEnglish: {
			Subject: "RYZE support replied", Heading: "New reply from RYZE support",
			Body:   "Hi{{if .Name}} {{.Name}}{{end}}, RYZE support replied to your conversation.",
			Action: "Read the reply", NeedsLink: true,
		},
		LocalePortuguese: {
			Subject: "O suporte RYZE respondeu", Heading: "Nova resposta do suporte RYZE",
			Body:   "Olá{{if .Name}} {{.Name}}{{end}}, o suporte RYZE respondeu à tua conversa.",
			Action: "Ler a resposta", NeedsLink: true,
		},
	},
}

// Data is the set of values a template may use. Link is an absolute URL to the
// RYZE web app built by the Mailer; Name and Program are plain text.
type Data struct {
	Name    string `json:"name"`
	Program string `json:"program,omitempty"`
	Link    string `json:"link,omitempty"`
	// Invoice is the purchase document an invoice_issued email carries.
	Invoice *InvoiceData `json:"invoice,omitempty"`
}

// InvoiceData is the snapshot of a purchase document shown in its email.
// Amounts are minor units; the email formats them in its own language.
type InvoiceData struct {
	Number        string `json:"number"`
	IssuedAt      string `json:"issued_at"` // YYYY-MM-DD
	Test          bool   `json:"test,omitempty"`
	SellerName    string `json:"seller_name"`
	SellerTaxID   string `json:"seller_tax_id,omitempty"`
	SellerAddress string `json:"seller_address,omitempty"`
	BuyerName     string `json:"buyer_name"`
	BuyerTaxID    string `json:"buyer_tax_id,omitempty"`
	Description   string `json:"description"`
	Quantity      int    `json:"quantity"`
	UnitPrice     int64  `json:"unit_price"`
	Tax           int64  `json:"tax"`
	TaxExemption  string `json:"tax_exemption,omitempty"`
	Total         int64  `json:"total"`
	Currency      string `json:"currency"`
}

// invoiceLabels is the wording of the document block, per language.
var invoiceLabels = map[string]map[string]string{
	LocaleEnglish: {
		"title": "Purchase receipt", "number": "Number", "date": "Date", "seller": "Seller", "buyer": "Customer",
		"taxID": "Taxpayer no.", "consumer": "Final consumer", "item": "Item", "qty": "Qty", "unit": "Unit price",
		"vat": "VAT", "total": "Total", "M10": "VAT exempt under article 53 of the Portuguese VAT Code (small-business scheme).",
		"noTax": "This document is a purchase receipt and is not a tax invoice.",
		"test":  "Test Mode purchase: no payment was taken.",
	},
	LocalePortuguese: {
		"title": "Comprovativo de compra", "number": "Número", "date": "Data", "seller": "Vendedor", "buyer": "Cliente",
		"taxID": "NIF", "consumer": "Consumidor final", "item": "Artigo", "qty": "Qtd.", "unit": "Preço unitário",
		"vat": "IVA", "total": "Total", "M10": "IVA - regime de isenção (artigo 53.º do CIVA).",
		"noTax": "Este documento é um comprovativo de compra e não tem valor fiscal.",
		"test":  "Compra do Modo de Teste: não foi cobrado nenhum pagamento.",
	},
}

// FormatMoney writes minor units in a language's style: "25,00 €" in
// Portuguese, "€25.00" in English (other currencies keep their code).
func FormatMoney(minorUnits int64, currency, locale string) string {
	sign := ""
	if minorUnits < 0 {
		sign = "-"
		minorUnits = -minorUnits
	}
	whole, cents := minorUnits/100, minorUnits%100
	symbol := strings.ToUpper(currency)
	if symbol == "EUR" {
		symbol = "€"
	}
	if NormalizeLocale(locale) == LocalePortuguese {
		return fmt.Sprintf("%s%d,%02d %s", sign, whole, cents, symbol)
	}
	if symbol == "€" {
		return fmt.Sprintf("%s€%d.%02d", sign, whole, cents)
	}
	return fmt.Sprintf("%s%d.%02d %s", sign, whole, cents, symbol)
}

// invoiceRows is the document block's content in one language.
type invoiceView struct {
	Labels                                        map[string]string
	Number, Date, Seller, SellerTaxID, SellerAddr string
	Buyer, BuyerTaxID, Description                string
	Quantity                                      int
	Unit, Tax, Total, Exemption                   string
	Test                                          bool
}

func newInvoiceView(data *InvoiceData, locale string) *invoiceView {
	if data == nil {
		return nil
	}
	labels := invoiceLabels[NormalizeLocale(locale)]
	view := &invoiceView{
		Labels: labels, Number: data.Number, Date: data.IssuedAt, Seller: data.SellerName, SellerTaxID: data.SellerTaxID,
		SellerAddr: data.SellerAddress, Buyer: data.BuyerName, BuyerTaxID: data.BuyerTaxID, Description: data.Description,
		Quantity: data.Quantity, Unit: FormatMoney(data.UnitPrice, data.Currency, locale), Tax: FormatMoney(data.Tax, data.Currency, locale),
		Total: FormatMoney(data.Total, data.Currency, locale), Test: data.Test,
	}
	if data.TaxExemption != "" {
		view.Exemption = labels[data.TaxExemption]
	}
	return view
}

var layout = htmltemplate.Must(htmltemplate.New("layout").Parse(`<!doctype html>
<html lang="{{.Lang}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Subject}}</title></head>
<body style="margin:0;background:#02041f;font-family:Arial,Helvetica,sans-serif;color:#ffffff">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#02041f;padding:32px 16px">
<tr><td align="center">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;background:#0a1846;border:1px solid #1d3a7a;border-radius:16px;padding:32px">
<tr><td style="font-size:22px;font-weight:800;letter-spacing:3px;color:#ffffff;padding-bottom:24px">RYZE</td></tr>
<tr><td style="font-size:24px;font-weight:700;color:#ffffff;padding-bottom:12px">{{.Heading}}</td></tr>
<tr><td style="font-size:16px;line-height:1.6;color:#c9d5ec;padding-bottom:24px">{{.Body}}</td></tr>
{{if .Link}}<tr><td style="padding-bottom:24px"><a href="{{.Link}}" style="display:inline-block;background:#00cbff;color:#02041f;text-decoration:none;font-weight:700;padding:14px 24px;border-radius:12px">{{.Action}}</a></td></tr>
<tr><td style="font-size:12px;line-height:1.5;color:#93a4c8;padding-bottom:16px;word-break:break-all">{{.Link}}</td></tr>{{end}}
{{with .Invoice}}<tr><td style="padding-bottom:24px">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:12px;color:#0b1638;font-size:14px;line-height:1.5">
<tr><td colspan="2" style="padding:20px 20px 4px;font-size:18px;font-weight:700">{{.Labels.title}}</td></tr>
<tr><td style="padding:0 20px;color:#4a5878">{{.Labels.number}}</td><td align="right" style="padding:0 20px;font-weight:700">{{.Number}}</td></tr>
<tr><td style="padding:0 20px 12px;color:#4a5878">{{.Labels.date}}</td><td align="right" style="padding:0 20px 12px">{{.Date}}</td></tr>
<tr><td colspan="2" style="padding:12px 20px;border-top:1px solid #dde3ee"><strong>{{.Labels.seller}}:</strong> {{.Seller}}{{if .SellerTaxID}} · {{.Labels.taxID}} {{.SellerTaxID}}{{end}}{{if .SellerAddr}}<br>{{.SellerAddr}}{{end}}<br>
<strong>{{.Labels.buyer}}:</strong> {{.Buyer}} · {{.Labels.taxID}} {{if .BuyerTaxID}}{{.BuyerTaxID}}{{else}}{{.Labels.consumer}}{{end}}</td></tr>
<tr><td style="padding:12px 20px 0;border-top:1px solid #dde3ee"><strong>{{.Description}}</strong><br><span style="color:#4a5878">{{.Labels.qty}} {{.Quantity}} × {{.Unit}}</span></td><td align="right" style="padding:12px 20px 0;border-top:1px solid #dde3ee">{{.Unit}}</td></tr>
<tr><td style="padding:4px 20px 0;color:#4a5878">{{.Labels.vat}}</td><td align="right" style="padding:4px 20px 0">{{.Tax}}</td></tr>
<tr><td style="padding:8px 20px 16px;font-size:16px;font-weight:700">{{.Labels.total}}</td><td align="right" style="padding:8px 20px 16px;font-size:16px;font-weight:700">{{.Total}}</td></tr>
<tr><td colspan="2" style="padding:12px 20px 20px;border-top:1px solid #dde3ee;font-size:12px;color:#4a5878">{{if .Exemption}}{{.Exemption}}<br>{{end}}{{if .Test}}{{.Labels.test}}<br>{{end}}{{.Labels.noTax}}</td></tr>
</table></td></tr>{{end}}
{{if .Footnote}}<tr><td style="font-size:13px;line-height:1.5;color:#93a4c8">{{.Footnote}}</td></tr>{{end}}
</table></td></tr></table></body></html>`))

// Render produces the subject, text and HTML of a template in a locale.
func Render(template, locale string, data Data) (Message, error) {
	byLocale, ok := copies[template]
	if !ok {
		return Message{}, fmt.Errorf("unknown email template %q", template)
	}
	wording, ok := byLocale[NormalizeLocale(locale)]
	if !ok {
		return Message{}, fmt.Errorf("template %q has no %q wording", template, locale)
	}
	if wording.NeedsLink && data.Link == "" {
		return Message{}, fmt.Errorf("template %q needs a link", template)
	}

	fill := func(value string) (string, error) {
		parsed, err := texttemplate.New("copy").Option("missingkey=error").Parse(value)
		if err != nil {
			return "", err
		}
		var out bytes.Buffer
		if err := parsed.Execute(&out, data); err != nil {
			return "", err
		}
		return out.String(), nil
	}

	subject, err := fill(wording.Subject)
	if err != nil {
		return Message{}, err
	}
	heading, err := fill(wording.Heading)
	if err != nil {
		return Message{}, err
	}
	body, err := fill(wording.Body)
	if err != nil {
		return Message{}, err
	}

	var html bytes.Buffer
	if err := layout.Execute(&html, map[string]any{
		"Lang": NormalizeLocale(locale), "Subject": subject, "Heading": heading, "Body": body,
		"Action": wording.Action, "Link": data.Link, "Footnote": wording.Footnote,
		"Invoice": newInvoiceView(data.Invoice, locale),
	}); err != nil {
		return Message{}, fmt.Errorf("render email layout: %w", err)
	}

	var text strings.Builder
	text.WriteString(heading + "\n\n" + body + "\n")
	if view := newInvoiceView(data.Invoice, locale); view != nil {
		l := view.Labels
		buyerTax := view.BuyerTaxID
		if buyerTax == "" {
			buyerTax = l["consumer"]
		}
		fmt.Fprintf(&text, "\n%s\n%s: %s\n%s: %s\n%s: %s\n%s: %s (%s %s)\n%s: %d × %s\n%s: %s\n%s: %s\n",
			l["title"], l["number"], view.Number, l["date"], view.Date, l["seller"], view.Seller,
			l["buyer"], view.Buyer, l["taxID"], buyerTax, view.Description, view.Quantity, view.Unit,
			l["vat"], view.Tax, l["total"], view.Total)
		if view.Exemption != "" {
			text.WriteString(view.Exemption + "\n")
		}
		if view.Test {
			text.WriteString(l["test"] + "\n")
		}
		text.WriteString(l["noTax"] + "\n")
	}
	if data.Link != "" {
		text.WriteString("\n" + wording.Action + ": " + data.Link + "\n")
	}
	if wording.Footnote != "" {
		text.WriteString("\n" + wording.Footnote + "\n")
	}
	text.WriteString("\nRYZE\n")

	return Message{Subject: subject, Text: text.String(), HTML: html.String()}, nil
}
