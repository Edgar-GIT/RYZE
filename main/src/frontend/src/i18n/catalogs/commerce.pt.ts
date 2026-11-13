import type { Catalog } from "@/i18n/i18n";

// Portuguese for the text shared by every purchase flow: the checkout states,
// the checkout hook's messages, program metadata chips and objectives.
export const COMMERCE_PT: Catalog = {
  // Checkout states and actions
  "You have access": "Já tens acesso",
  "You already own this training plan.": "Este plano de treino já é teu.",
  "Open program": "Abrir programa",
  "Back to Training plans": "Voltar aos planos de treino",
  "Back to Services": "Voltar aos serviços",
  "Checking access…": "A verificar o acesso…",
  "Sign in to purchase this training plan.": "Entra na tua conta para comprar este plano de treino.",
  "Sign in": "Entrar",
  "Sign in to continue": "Entra na tua conta para continuar",
  "Preparing secure checkout…": "A preparar o pagamento seguro…",
  "Redirecting to the secure payment page. Do not close this window…":
    "A redirecionar para a página de pagamento seguro. Não feches esta janela…",
  "Confirming your payment. This may take a few seconds…": "A confirmar o teu pagamento. Pode demorar alguns segundos…",
  "Purchase complete": "Compra concluída",
  "Purchase confirmed": "Compra confirmada",
  "You now have access to this training plan.": "Já tens acesso a este plano de treino.",
  "My Programs": "Os meus programas",
  "Payment cancelled": "Pagamento cancelado",
  "No payment was completed. You can try again whenever you are ready.":
    "Nenhum pagamento foi concluído. Podes tentar novamente quando quiseres.",
  "Try again": "Tentar novamente",
  Retry: "Tentar novamente",
  "Purchase unavailable": "Compra indisponível",
  "Test Mode": "Modo de teste",
  "Test Mode purchase — this plan is granted instantly at no cost. No real payment is made.":
    "Compra em modo de teste — o plano é atribuído de imediato e sem custo. Não é feito nenhum pagamento real.",
  "Get plan in Test Mode": "Obter plano em modo de teste",
  "Checkout unavailable": "Pagamento indisponível",
  "No payment provider is currently configured. Please try again later.":
    "Neste momento não há nenhum fornecedor de pagamento configurado. Tenta mais tarde.",
  "One-time purchase. Secure checkout.": "Compra única. Pagamento seguro.",
  "Payment method": "Método de pagamento",
  "Buy now": "Comprar agora",
  "Resume checkout": "Retomar pagamento",
  "Checkout is temporarily unavailable. No payment can be started right now.":
    "O pagamento está temporariamente indisponível. Neste momento não é possível iniciar nenhum pagamento.",
  "Test Mode is active, so this purchase completes without contacting any payment provider.":
    "O modo de teste está ativo: esta compra é concluída sem contactar nenhum fornecedor de pagamento.",

  // use_program_checkout messages
  "Your payment is being confirmed. If your access does not appear within a few minutes, please try again later.":
    "O teu pagamento está a ser confirmado. Se o acesso não aparecer dentro de alguns minutos, tenta mais tarde.",
  "We could not confirm your payment.": "Não foi possível confirmar o teu pagamento.",
  "Checkout is temporarily unavailable. Please try again later.": "O pagamento está temporariamente indisponível. Tenta mais tarde.",
  "Checkout is temporarily unavailable. Please try again.": "O pagamento está temporariamente indisponível. Tenta novamente.",
  "This purchase was already started with another payment method. Please try again later.":
    "Esta compra já foi iniciada com outro método de pagamento. Tenta mais tarde.",
  "We could not start the checkout. No money was taken at this point.":
    "Não foi possível iniciar o pagamento. Não foi cobrado qualquer valor.",
  "We could not complete this Test Mode purchase. Please try again.":
    "Não foi possível concluir esta compra em modo de teste. Tenta novamente.",
  "We could not start your purchase. No money was taken at this point.":
    "Não foi possível iniciar a tua compra. Não foi cobrado qualquer valor.",
  "We could not verify your access. Please try again.": "Não foi possível verificar o teu acesso. Tenta novamente.",

  // Program metadata
  "{count} week": "{count} semana",
  "{count} weeks": "{count} semanas",
  "{count} day/week": "{count} dia/semana",
  "{count} days/week": "{count} dias/semana",
  Hypertrophy: "Hipertrofia",
  Strength: "Força",
  "Fat Loss": "Perda de gordura",
  "At Home": "Em casa",
  Beginner: "Iniciante",
  Intermediate: "Intermédio",
  Advanced: "Avançado",
  "Gain muscle": "Ganhar massa muscular",
  "Lose fat": "Perder gordura",
  "Strength & conditioning": "Força e condição física",
  "Training + nutrition": "Treino + nutrição",

  // Premium Level 1 nutrition errors (services/premium_level1_api.ts)
  "This programme's food list cannot cover every restriction you recorded. Your questionnaire, your plan and your purchase are all unchanged.":
    "A lista de alimentos deste programa não cobre todas as restrições que indicaste. O teu questionário, o teu plano e a tua compra ficam iguais.",
  "We could not build your nutrition programme yet. Your purchase is unaffected — please try again.":
    "Ainda não foi possível criar o teu programa de nutrição. A tua compra não é afetada — tenta novamente."
};
