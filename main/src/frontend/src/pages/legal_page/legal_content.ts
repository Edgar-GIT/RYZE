import { CNIACC_URL, CNPD_URL, COMPLAINTS_BOOK_URL, CONSUMER_PORTAL_URL, LEGAL_ENTITY as E } from "@/constants/legal";

/*
 * RYZE legal documents (owner request, 2026-11-23: Terms and Conditions,
 * Privacy Policy, Cookie Policy and Legal notice, in line with the GDPR and EU
 * consumer law). They describe what the code base actually does: the
 * cookies and storage listed are the ones the application sets; the
 * processors are the ones it integrates with. Company identity fields come
 * from constants/legal.ts. A legal review is recommended before launch.
 */

export type LegalDocId = "termos" | "privacidade" | "cookies" | "aviso-legal";

export interface LegalSection {
  id: string;
  title: string;
  paragraphs?: string[];
  list?: string[];
  table?: { headers: string[]; rows: string[][] };
}

export interface LegalDoc {
  title: string;
  summary: string;
  sections: LegalSection[];
}

const entity = `${E.name}, NIPC ${E.taxId}, com sede em ${E.address}`;
const entityEn = `${E.name}, company number ${E.taxId}, registered office at ${E.address}`;

const COOKIE_ROWS_PT: string[][] = [
  ["ryze_access_token", "Cookie (RYZE)", "Mantém a tua sessão iniciada de forma segura (HttpOnly).", "Estritamente necessário", "Até terminares a sessão ou expirar"],
  ["ryze_google_state", "Cookie (RYZE)", "Protege o início de sessão com Google contra pedidos forjados.", "Estritamente necessário", "Minutos"],
  ["ryze_admin_access_token, ryze_admin_stage_token", "Cookie (RYZE)", "Sessão da área de administração (só equipa RYZE).", "Estritamente necessário", "Sessão"],
  ["ryze_test_session", "Cookie (RYZE)", "Modo de Teste interno da equipa RYZE.", "Estritamente necessário", "Sessão"],
  ["ryze.locale", "Armazenamento local", "Guarda o idioma que escolheste (PT/EN).", "Funcional (pedido por ti)", "Até o apagares"],
  ["ryze.planPreferences", "Armazenamento local", "Guarda as preferências de planos que escolheste.", "Funcional (pedido por ti)", "Até o apagares"],
  ["ryze.cookieConsent", "Armazenamento local", "Guarda a tua escolha sobre cookies.", "Estritamente necessário", "12 meses"],
  ["ryze.navigationStack, ryze.afterLogin, ryze.levelUpSeen.*", "Armazenamento de sessão", "Botão voltar, regresso após login e animações já vistas.", "Estritamente necessário", "Até fechares o separador"],
  ["__stripe_mid, __stripe_sid", "Cookie (Stripe)", "Prevenção de fraude no pagamento, apenas quando pagas com Stripe.", "Estritamente necessário (pagamento)", "Até 1 ano / 30 minutos"]
];

const COOKIE_ROWS_EN: string[][] = [
  ["ryze_access_token", "Cookie (RYZE)", "Keeps you signed in securely (HttpOnly).", "Strictly necessary", "Until you sign out or it expires"],
  ["ryze_google_state", "Cookie (RYZE)", "Protects Google sign-in against forged requests.", "Strictly necessary", "Minutes"],
  ["ryze_admin_access_token, ryze_admin_stage_token", "Cookie (RYZE)", "Administration area session (RYZE staff only).", "Strictly necessary", "Session"],
  ["ryze_test_session", "Cookie (RYZE)", "Internal Test Mode for RYZE staff.", "Strictly necessary", "Session"],
  ["ryze.locale", "Local storage", "Remembers the language you chose (PT/EN).", "Functional (requested by you)", "Until you clear it"],
  ["ryze.planPreferences", "Local storage", "Remembers the plan preferences you chose.", "Functional (requested by you)", "Until you clear it"],
  ["ryze.cookieConsent", "Local storage", "Remembers your cookie choice.", "Strictly necessary", "12 months"],
  ["ryze.navigationStack, ryze.afterLogin, ryze.levelUpSeen.*", "Session storage", "Back button, return after sign-in and animations already seen.", "Strictly necessary", "Until you close the tab"],
  ["__stripe_mid, __stripe_sid", "Cookie (Stripe)", "Payment fraud prevention, only when you pay with Stripe.", "Strictly necessary (payment)", "Up to 1 year / 30 minutes"]
];

const PT: Record<LegalDocId, LegalDoc> = {
  termos: {
    title: "Termos e Condições",
    summary: "As regras de utilização da RYZE e das compras de planos, incluindo os teus direitos como consumidor.",
    sections: [
      {
        id: "quem-somos",
        title: "1. Quem somos",
        paragraphs: [
          `A RYZE é uma plataforma de treino e nutrição explorada por ${entity}, matriculada na ${E.registry} (a "RYZE"). Contacto: ${E.email}.`,
          "Estes Termos aplicam-se a quem visita o site, cria conta, usa os planos ou compra produtos na RYZE. Ao criar conta ou comprar, aceitas estes Termos e a Política de Privacidade."
        ]
      },
      {
        id: "servico",
        title: "2. O serviço",
        list: [
          "Planos de treino gratuitos, que podes ativar na tua biblioteca (podes ter vários ativos ao mesmo tempo).",
          "Premium Nível 1: um pacote de treino e nutrição escolhido do catálogo RYZE para o teu objetivo.",
          "Premium Nível 2: um plano de treino e nutrição preparado só para ti, a partir das tuas respostas ao questionário, por um treinador RYZE ou pela própria RYZE. Depois do pagamento e do questionário, aguardas que o plano seja preparado; quando é entregue recebes uma notificação e ele aparece na tua área pessoal.",
          "Registo de treinos, progresso, recuperação, conquistas, lembretes de treino, perfil público e feedback.",
          "Não podes ter o Premium Nível 1 e o Nível 2 ativos ao mesmo tempo."
        ]
      },
      {
        id: "conta",
        title: "3. Conta",
        paragraphs: [
          `Para criar conta tens de ter pelo menos ${E.minimumAge} anos. Se fores menor de idade, só podes fazer compras com autorização de quem exerce as responsabilidades parentais.`,
          "És responsável por manter a tua password confidencial e pela atividade na tua conta. O teu @nome de utilizador é único e público; a alcunha e a foto de perfil também são visíveis no teu perfil público.",
          "Podes apagar a tua conta a qualquer momento nas definições da conta."
        ]
      },
      {
        id: "saude",
        title: "4. Saúde e segurança",
        paragraphs: [
          "Os planos, a nutrição, as estimativas (como a recuperação muscular ou a 1RM estimada) e os conteúdos da RYZE são informação geral de treino e não substituem aconselhamento médico, nutricional ou de outro profissional de saúde.",
          "Consulta um médico antes de começares um programa de exercício ou de alterares a alimentação, sobretudo se tiveres alguma condição de saúde, lesão, gravidez ou estiveres a tomar medicação. Pára o exercício se sentires dor, tonturas ou mal-estar."
        ]
      },
      {
        id: "precos",
        title: "5. Preços e pagamentos",
        paragraphs: [
          "Os preços são apresentados em euros, são finais e incluem os impostos aplicáveis. Cada compra é um pagamento único, sem subscrição nem renovação automática.",
          "Os pagamentos são processados por prestadores de serviços de pagamento (Stripe para cartão e MB WAY; PayPal). A RYZE não recebe nem guarda os dados do teu cartão.",
          "Depois do pagamento confirmado recebes o acesso na tua conta e um comprovativo de compra. Podes indicar o teu NIF para constar no comprovativo.",
          "Códigos de desconto: cada código tem uma percentagem e um prazo (no máximo 30 dias), pode ter um número limitado de utilizações, só pode ser usado uma vez por pessoa, aplica-se antes do pagamento e não é convertível em dinheiro."
        ]
      },
      {
        id: "livre-resolucao",
        title: "6. Direito de livre resolução (desistência)",
        paragraphs: [
          "Como consumidor, tens em regra 14 dias para desistir de um contrato celebrado à distância, sem indicar motivo.",
          "Os planos da RYZE são conteúdo digital fornecido de imediato. Antes de pagares, pedimos-te que solicites expressamente o acesso imediato e que reconheças que, com o início do acesso, perdes o direito de livre resolução (Diretiva 2011/83/UE, art. 16.º, alínea m), e Decreto-Lei n.º 24/2014, art. 17.º). A RYZE guarda o registo dessa aceitação.",
          "No Premium Nível 2, o plano é preparado à tua medida. Se, por algum motivo, o direito de livre resolução se aplicar, podes exercê-lo através do email indicado acima ou do formulário legal de livre resolução; reembolsamos pelo mesmo meio de pagamento no prazo de 14 dias.",
          "Quando há reembolso (por desistência ou por o conteúdo não estar conforme), o acesso ao plano comprado — e, no Premium Nível 2, ao plano pessoal entregue — termina no momento do reembolso."
        ]
      },
      {
        id: "conformidade",
        title: "7. Conformidade e garantia do conteúdo digital",
        paragraphs: [
          "O conteúdo digital tem de corresponder ao descrito e funcionar como indicado (Decreto-Lei n.º 84/2021). Se algo não estiver conforme, contacta-nos: corrigimos, substituímos ou, quando a lei o preveja, reduzimos o preço ou reembolsamos."
        ]
      },
      {
        id: "conteudos-utilizador",
        title: "8. Os teus conteúdos",
        paragraphs: [
          "Continuas dono do que publicas (feedback, fotos, biografia, redes sociais). Ao publicar, autorizas a RYZE a mostrar esse conteúdo na plataforma enquanto estiver publicado. O feedback só aparece depois de revisto pela equipa RYZE, e as fotos dos teus treinos são privadas.",
          "Não podes publicar conteúdos ilegais, ofensivos, que violem direitos de terceiros ou que não sejam teus. Podemos recusar ou remover esses conteúdos."
        ]
      },
      {
        id: "treinadores",
        title: "9. Treinadores",
        paragraphs: [
          "Os treinadores RYZE são profissionais que preparam planos através da plataforma. A candidatura, a verificação de documentos e a relação comercial entre o treinador e a RYZE regem-se por condições próprias.",
          "A RYZE é uma experiência autónoma: não há contacto direto entre clientes e treinadores. Para falar com a RYZE usa a página de contactos."
        ]
      },
      {
        id: "utilizacao",
        title: "10. Utilização aceitável",
        list: [
          "Não partilhes a tua conta nem os planos comprados com terceiros.",
          "Não copies, revendas ou distribuas os conteúdos da RYZE.",
          "Não tentes aceder a dados de outras pessoas, contornar a segurança ou sobrecarregar o serviço.",
          "Podemos suspender ou encerrar contas que violem estes Termos, depois de te avisarmos sempre que a lei o permita."
        ]
      },
      {
        id: "propriedade",
        title: "11. Propriedade intelectual",
        paragraphs: [
          "A marca, o logótipo, o design, os planos, os textos e o software da RYZE pertencem à RYZE ou aos seus licenciadores. A compra dá-te uma licença pessoal e não transmissível para usar o conteúdo na tua conta."
        ]
      },
      {
        id: "responsabilidade",
        title: "12. Responsabilidade",
        paragraphs: [
          "A RYZE presta o serviço com cuidado, mas não pode garantir resultados físicos concretos, que dependem de ti. Nada nestes Termos limita a responsabilidade que a lei não permite limitar, incluindo por dolo, culpa grave ou danos à vida e à saúde."
        ]
      },
      {
        id: "alteracoes",
        title: "13. Alterações",
        paragraphs: [
          "Podemos atualizar estes Termos. Avisamos com antecedência as alterações importantes; as compras já feitas continuam sujeitas à versão em vigor na data da compra."
        ]
      },
      {
        id: "lei",
        title: "14. Lei aplicável e litígios",
        paragraphs: [
          "Aplica-se a lei portuguesa, sem prejuízo das normas imperativas de proteção do consumidor do país onde resides.",
          `Em caso de litígio de consumo, podes recorrer ao Livro de Reclamações Eletrónico (${COMPLAINTS_BOOK_URL}) e a uma entidade de resolução alternativa de litígios, como o CNIACC (${CNIACC_URL}); a lista completa está em ${CONSUMER_PORTAL_URL}.`
        ]
      }
    ]
  },
  privacidade: {
    title: "Política de Privacidade",
    summary: "Que dados pessoais tratamos, para quê, com que base legal, durante quanto tempo e quais são os teus direitos (RGPD).",
    sections: [
      {
        id: "responsavel",
        title: "1. Responsável pelo tratamento",
        paragraphs: [`O responsável pelo tratamento dos teus dados é ${entity}. Para qualquer questão de privacidade: ${E.privacyEmail}.`]
      },
      {
        id: "dados",
        title: "2. Que dados tratamos",
        list: [
          "Conta: email, password (guardada cifrada), nome, @nome de utilizador, alcunha e idioma.",
          "Perfil público (opcional): foto, biografia, idade calculada a partir da data de nascimento, anos de treino, peso e redes sociais.",
          "Treino: planos ativos, sessões, séries, pesos, repetições, fotos dos treinos (privadas), conquistas e rankings agregados.",
          "Dados de saúde (opcionais): medições corporais, peso e respostas aos questionários do Premium (por exemplo, objetivos, lesões ou condições). São dados de categoria especial e só os tratamos com o teu consentimento explícito, que dás ao guardar a primeira medição ou ao enviar o questionário, e que podes retirar em Conta → Privacidade e notificações (isso apaga esses dados).",
          "Compras: produto, preço, método de pagamento, estado, NIF (opcional) e as tuas aceitações no checkout. Os dados do cartão ficam só no prestador de pagamento.",
          "Comunicações: feedback e respetivas imagens, notificações e lembretes de treino.",
          "Treinadores: dados da candidatura e documentos comprovativos.",
          "Dados técnicos: endereço IP e registos de segurança, cookies estritamente necessários."
        ]
      },
      {
        id: "finalidades",
        title: "3. Finalidades e bases legais",
        table: {
          headers: ["Finalidade", "Base legal (RGPD)"],
          rows: [
            ["Criar e gerir a conta, fornecer os planos e as funcionalidades", "Execução do contrato (art. 6.º, n.º 1, al. b)"],
            ["Processar pagamentos e emitir comprovativos", "Execução do contrato e obrigação legal (al. b) e c))"],
            ["Tratar dados de saúde que indicas (medições, questionários)", "Consentimento explícito (art. 9.º, n.º 2, al. a)), que podes retirar a qualquer momento"],
            ["Mostrar o teu perfil público, feedback e rankings agregados", "Execução do contrato e o teu consentimento quando publicas"],
            ["Emails transacionais (verificação, compras, entrega do Premium)", "Execução do contrato"],
            ["Lembretes de treino ocasionais (podes desligá-los na conta)", "Interesse legítimo (al. f))"],
            ["Segurança, prevenção de fraude e abuso", "Interesse legítimo (al. f))"],
            ["Cumprir obrigações fiscais e legais", "Obrigação legal (al. c))"]
          ]
        }
      },
      {
        id: "destinatarios",
        title: "4. Com quem partilhamos",
        list: [
          "Prestadores de pagamento: Stripe (cartão e MB WAY) e PayPal, como responsáveis pelos dados de pagamento.",
          "Google, se escolheres iniciar sessão com Google.",
          "No Premium Nível 2, o treinador que aceita o teu pedido (ou a RYZE), apenas com os dados necessários para preparar o teu plano.",
          "Prestadores de alojamento e de envio de email, como subcontratantes ao abrigo de contrato (art. 28.º RGPD).",
          "Autoridades, quando a lei o exigir.",
          "Não vendemos os teus dados nem os usamos para publicidade."
        ]
      },
      {
        id: "transferencias",
        title: "5. Transferências internacionais",
        paragraphs: [
          "Alguns prestadores (como Stripe, PayPal ou Google) podem tratar dados fora do Espaço Económico Europeu. Nesses casos, a transferência assenta numa decisão de adequação (por exemplo, o Quadro de Privacidade de Dados UE-EUA) ou em cláusulas contratuais-tipo aprovadas pela Comissão Europeia."
        ]
      },
      {
        id: "conservacao",
        title: "6. Durante quanto tempo",
        list: [
          "Dados da conta e de treino: enquanto tiveres conta. Quando apagas a conta, durante 30 dias podes recuperá-la voltando a registar-te com o mesmo email; depois disso os dados que te identificam são apagados e os registos de treino ficam anónimos.",
          "Compras e comprovativos: pelo prazo legal de conservação de documentos fiscais (em regra 10 anos).",
          "Registos de segurança: o tempo necessário à segurança do serviço.",
          "Candidaturas de treinador recusadas: o tempo necessário para a decisão e eventuais reclamações."
        ]
      },
      {
        id: "direitos",
        title: "7. Os teus direitos",
        paragraphs: [
          "Podes pedir acesso, retificação, apagamento, limitação do tratamento, portabilidade e opor-te ao tratamento baseado em interesse legítimo. Podes retirar o consentimento a qualquer momento, sem afetar o tratamento já feito.",
          `Muitos destes direitos estão disponíveis diretamente na tua conta (editar perfil, apagar fotos, retirar o consentimento para dados de saúde, apagar a conta). Para o resto, escreve para ${E.privacyEmail}; respondemos no prazo de um mês.`,
          `Tens também o direito de apresentar reclamação à Comissão Nacional de Proteção de Dados (${CNPD_URL}).`
        ]
      },
      {
        id: "decisoes",
        title: "8. Decisões automatizadas",
        paragraphs: [
          "A escolha de um pacote do Premium Nível 1 faz-se com base no objetivo e no sexo que indicas, entre os pacotes publicados, e as estimativas de treino são calculadas automaticamente. Nenhuma destas operações produz efeitos jurídicos nem te afeta de forma significativa semelhante."
        ]
      },
      {
        id: "seguranca",
        title: "9. Segurança",
        paragraphs: [
          "Usamos ligações cifradas, passwords guardadas com algoritmos de hash, cookies de sessão HttpOnly, controlo de acessos e registos de auditoria. As fotos são re-codificadas sem metadados (como a localização)."
        ]
      },
      {
        id: "menores",
        title: "10. Menores",
        paragraphs: [`A RYZE não se destina a menores de ${E.minimumAge} anos.`]
      },
      {
        id: "alteracoes-privacidade",
        title: "11. Alterações",
        paragraphs: ["Se alterarmos esta política de forma relevante, avisamos-te na plataforma ou por email."]
      }
    ]
  },
  cookies: {
    title: "Política de Cookies",
    summary: "Os cookies e o armazenamento local que a RYZE usa, para que servem e como podes gerir a tua escolha.",
    sections: [
      {
        id: "o-que-sao",
        title: "1. O que são cookies",
        paragraphs: [
          "Cookies são pequenos ficheiros guardados pelo browser. Usamos também o armazenamento local e de sessão do browser, que funcionam de forma semelhante."
        ]
      },
      {
        id: "que-usamos",
        title: "2. Que cookies usamos",
        paragraphs: [
          "A RYZE só usa cookies e armazenamento estritamente necessários ao funcionamento do serviço e funcionais que tu próprio pedes (como o idioma). Não usamos cookies de análise estatística nem de publicidade."
        ],
        table: { headers: ["Nome", "Tipo", "Para que serve", "Categoria", "Duração"], rows: COOKIE_ROWS_PT }
      },
      {
        id: "consentimento",
        title: "3. Consentimento",
        paragraphs: [
          "Os cookies estritamente necessários não precisam de consentimento (Lei n.º 41/2004, art. 5.º, e Diretiva 2002/58/CE). Se no futuro passarmos a usar cookies de análise ou publicidade, só os ativaremos com o teu consentimento prévio, que podes dar ou recusar no aviso de cookies.",
          "Podes rever a tua escolha a qualquer momento no botão \"Definições de cookies\" nesta página."
        ]
      },
      {
        id: "gerir",
        title: "4. Gerir no browser",
        paragraphs: [
          "Também podes apagar ou bloquear cookies nas definições do teu browser. Se bloqueares os cookies estritamente necessários, não conseguirás iniciar sessão nem pagar."
        ]
      }
    ]
  },
  "aviso-legal": {
    title: "Aviso Legal",
    summary: "A identificação da entidade que explora a RYZE e as informações legais obrigatórias.",
    sections: [
      {
        id: "identificacao",
        title: "1. Identificação",
        list: [
          `Denominação: ${E.name}`,
          `NIPC: ${E.taxId}`,
          `Sede: ${E.address}`,
          `Registo: ${E.registry}`,
          `Capital social: ${E.shareCapital}`,
          `Email: ${E.email}`
        ]
      },
      {
        id: "reclamacoes",
        title: "2. Livro de Reclamações",
        paragraphs: [`Podes apresentar uma reclamação no Livro de Reclamações Eletrónico: ${COMPLAINTS_BOOK_URL}.`]
      },
      {
        id: "ral",
        title: "3. Resolução alternativa de litígios",
        paragraphs: [
          `Em caso de litígio de consumo, podes recorrer a uma entidade de resolução alternativa de litígios, como o CNIACC — Centro Nacional de Informação e Arbitragem de Conflitos de Consumo (${CNIACC_URL}). Mais informação no Portal do Consumidor: ${CONSUMER_PORTAL_URL}.`
        ]
      },
      {
        id: "documentos",
        title: "4. Outros documentos",
        paragraphs: ["Consulta também os Termos e Condições, a Política de Privacidade e a Política de Cookies."]
      }
    ]
  }
};

const EN: Record<LegalDocId, LegalDoc> = {
  termos: {
    title: "Terms and Conditions",
    summary: "The rules for using RYZE and buying plans, including your rights as a consumer.",
    sections: [
      {
        id: "quem-somos",
        title: "1. Who we are",
        paragraphs: [
          `RYZE is a training and nutrition platform operated by ${entityEn}, registered at ${E.registry} ("RYZE"). Contact: ${E.email}.`,
          "These Terms apply to anyone who visits the site, creates an account, uses the plans or buys products on RYZE. By creating an account or buying, you accept these Terms and the Privacy Policy."
        ]
      },
      {
        id: "servico",
        title: "2. The service",
        list: [
          "Free training plans you can add to your library (several can be active at once).",
          "Premium Level 1: a training and nutrition package chosen from the RYZE catalogue for your goal.",
          "Premium Level 2: a training and nutrition plan made only for you from your questionnaire answers, by a RYZE trainer or by RYZE itself. After paying and answering, you wait while it is prepared; when it is delivered you are notified and it appears in your personal area.",
          "Workout logging, progress, recovery, achievements, training reminders, public profile and feedback.",
          "Premium Level 1 and Level 2 cannot be active at the same time."
        ]
      },
      {
        id: "conta",
        title: "3. Account",
        paragraphs: [
          `You must be at least ${E.minimumAge} to create an account. Minors may only buy with the consent of whoever holds parental responsibility.`,
          "You keep your password confidential and are responsible for activity on your account. Your @username is unique and public; your nickname and profile photo are also shown on your public profile.",
          "You can delete your account at any time in your account settings."
        ]
      },
      {
        id: "saude",
        title: "4. Health and safety",
        paragraphs: [
          "RYZE plans, nutrition, estimates (such as muscle recovery or estimated 1RM) and content are general training information and do not replace medical, nutritional or other health advice.",
          "See a doctor before starting an exercise programme or changing your diet, especially if you have a health condition, injury, are pregnant or take medication. Stop exercising if you feel pain, dizziness or unwell."
        ]
      },
      {
        id: "precos",
        title: "5. Prices and payments",
        paragraphs: [
          "Prices are shown in euros, are final and include applicable taxes. Each purchase is a one-time payment, with no subscription or automatic renewal.",
          "Payments are processed by payment service providers (Stripe for card and MB WAY; PayPal). RYZE never receives or stores your card details.",
          "Once the payment is confirmed you get access in your account and a purchase receipt. You can add your tax number to the receipt.",
          "Discount codes: each code has a percentage and a validity (at most 30 days), may have a limited number of uses, can be used once per person, applies before payment and cannot be exchanged for cash."
        ]
      },
      {
        id: "livre-resolucao",
        title: "6. Right of withdrawal",
        paragraphs: [
          "As a consumer you generally have 14 days to withdraw from a distance contract without giving a reason.",
          "RYZE plans are digital content supplied immediately. Before paying, we ask you to expressly request immediate access and acknowledge that, once access starts, you lose your right of withdrawal (Directive 2011/83/EU, art. 16(m); Portuguese Decree-Law 24/2014, art. 17). RYZE keeps a record of that acceptance.",
          "Premium Level 2 is prepared for you. Where the right of withdrawal applies, you can exercise it by email at the address above or with the statutory withdrawal form; we refund through the same payment method within 14 days.",
          "When a purchase is refunded (withdrawal or non-conformity), access to the plan — and, for Premium Level 2, to the delivered personal plan — ends at the time of the refund."
        ]
      },
      {
        id: "conformidade",
        title: "7. Conformity of digital content",
        paragraphs: [
          "Digital content must match its description and work as stated. If it does not, contact us: we fix or replace it or, where the law provides, reduce the price or refund you."
        ]
      },
      {
        id: "conteudos-utilizador",
        title: "8. Your content",
        paragraphs: [
          "You keep ownership of what you publish (feedback, photos, bio, social links). By publishing, you allow RYZE to show it on the platform while it is published. Feedback only appears after review by the RYZE team, and your workout photos are private.",
          "You may not publish illegal or offensive content, content that infringes others' rights or that is not yours. We may refuse or remove such content."
        ]
      },
      {
        id: "treinadores",
        title: "9. Trainers",
        paragraphs: [
          "RYZE trainers are professionals who prepare plans through the platform. Applications, document checks and the commercial relationship between a trainer and RYZE are governed by separate terms.",
          "RYZE is a self-service experience: clients and trainers do not contact each other directly. To reach RYZE, use the contact page."
        ]
      },
      {
        id: "utilizacao",
        title: "10. Acceptable use",
        list: [
          "Do not share your account or purchased plans with others.",
          "Do not copy, resell or distribute RYZE content.",
          "Do not try to access other people's data, bypass security or overload the service.",
          "We may suspend or close accounts that breach these Terms, after notice wherever the law allows."
        ]
      },
      {
        id: "propriedade",
        title: "11. Intellectual property",
        paragraphs: [
          "The RYZE brand, logo, design, plans, texts and software belong to RYZE or its licensors. A purchase grants you a personal, non-transferable licence to use the content in your account."
        ]
      },
      {
        id: "responsabilidade",
        title: "12. Liability",
        paragraphs: [
          "RYZE provides the service with care but cannot guarantee specific physical results, which depend on you. Nothing in these Terms limits liability that cannot be limited by law, including for wilful misconduct, gross negligence or harm to life and health."
        ]
      },
      {
        id: "alteracoes",
        title: "13. Changes",
        paragraphs: ["We may update these Terms. We give notice of important changes; purchases already made remain under the version in force when you bought."]
      },
      {
        id: "lei",
        title: "14. Governing law and disputes",
        paragraphs: [
          "Portuguese law applies, without prejudice to the mandatory consumer protection rules of your country of residence.",
          `For consumer disputes you can use the electronic complaints book (${COMPLAINTS_BOOK_URL}) and an alternative dispute resolution body such as CNIACC (${CNIACC_URL}); the full list is at ${CONSUMER_PORTAL_URL}.`
        ]
      }
    ]
  },
  privacidade: {
    title: "Privacy Policy",
    summary: "Which personal data we process, why, on what legal basis, for how long, and your rights under the GDPR.",
    sections: [
      { id: "responsavel", title: "1. Controller", paragraphs: [`The controller of your data is ${entityEn}. For any privacy question: ${E.privacyEmail}.`] },
      {
        id: "dados",
        title: "2. Data we process",
        list: [
          "Account: email, password (stored hashed), name, @username, nickname and language.",
          "Public profile (optional): photo, bio, age calculated from your date of birth, years of training, weight and social links.",
          "Training: active plans, sessions, sets, weights, reps, workout photos (private), achievements and aggregate rankings.",
          "Health data (optional): body measurements, weight and Premium questionnaire answers (for example goals, injuries or conditions). These are special category data, processed only with your explicit consent, given when you save your first measurement or send the questionnaire; you can withdraw it in Account → Privacy and notifications, which erases that data.",
          "Purchases: product, price, payment method, status, tax number (optional) and your checkout acceptances. Card details stay with the payment provider.",
          "Communications: feedback and its pictures, notifications and training reminders.",
          "Trainers: application data and supporting documents.",
          "Technical data: IP address, security logs and strictly necessary cookies."
        ]
      },
      {
        id: "finalidades",
        title: "3. Purposes and legal bases",
        table: {
          headers: ["Purpose", "Legal basis (GDPR)"],
          rows: [
            ["Create and run your account, provide plans and features", "Performance of the contract (art. 6(1)(b))"],
            ["Process payments and issue receipts", "Contract and legal obligation (art. 6(1)(b), (c))"],
            ["Process health data you provide (measurements, questionnaires)", "Explicit consent (art. 9(2)(a)), which you can withdraw at any time"],
            ["Show your public profile, feedback and aggregate rankings", "Contract and your consent when you publish"],
            ["Transactional emails (verification, purchases, Premium delivery)", "Performance of the contract"],
            ["Occasional training reminders (you can turn them off in your account)", "Legitimate interest (art. 6(1)(f))"],
            ["Security, fraud and abuse prevention", "Legitimate interest (art. 6(1)(f))"],
            ["Tax and legal obligations", "Legal obligation (art. 6(1)(c))"]
          ]
        }
      },
      {
        id: "destinatarios",
        title: "4. Who we share with",
        list: [
          "Payment providers: Stripe (card and MB WAY) and PayPal, as controllers of payment data.",
          "Google, if you choose to sign in with Google.",
          "For Premium Level 2, the trainer who accepts your request (or RYZE), only with the data needed to prepare your plan.",
          "Hosting and email providers, as processors under contract (GDPR art. 28).",
          "Authorities, where the law requires.",
          "We do not sell your data or use it for advertising."
        ]
      },
      {
        id: "transferencias",
        title: "5. International transfers",
        paragraphs: [
          "Some providers (such as Stripe, PayPal or Google) may process data outside the European Economic Area. Those transfers rely on an adequacy decision (for example the EU-US Data Privacy Framework) or on the European Commission's standard contractual clauses."
        ]
      },
      {
        id: "conservacao",
        title: "6. Retention",
        list: [
          "Account and training data: while you have an account. When you delete it, you can restore it for 30 days by registering again with the same email; after that the data that identifies you is erased and training records are kept anonymous.",
          "Purchases and receipts: for the legal retention period for tax documents (usually 10 years).",
          "Security logs: as long as needed for the security of the service.",
          "Rejected trainer applications: as long as needed for the decision and any complaint."
        ]
      },
      {
        id: "direitos",
        title: "7. Your rights",
        paragraphs: [
          "You can ask for access, rectification, erasure, restriction, portability and object to processing based on legitimate interest. You can withdraw consent at any time without affecting processing already done.",
          `Many of these rights are available directly in your account (edit profile, delete photos, withdraw consent for health data, delete the account). For the rest, write to ${E.privacyEmail}; we answer within one month.`,
          `You can also lodge a complaint with the Portuguese data protection authority, CNPD (${CNPD_URL}).`
        ]
      },
      {
        id: "decisoes",
        title: "8. Automated decisions",
        paragraphs: [
          "Choosing a Premium Level 1 package uses the goal and sex you give, among the published packages, and training estimates are calculated automatically. None of these produce legal effects or similarly significantly affect you."
        ]
      },
      {
        id: "seguranca",
        title: "9. Security",
        paragraphs: [
          "We use encrypted connections, hashed passwords, HttpOnly session cookies, access control and audit logs. Photos are re-encoded without metadata (such as location)."
        ]
      },
      { id: "menores", title: "10. Minors", paragraphs: [`RYZE is not intended for people under ${E.minimumAge}.`] },
      { id: "alteracoes-privacidade", title: "11. Changes", paragraphs: ["If we change this policy in a relevant way, we will tell you on the platform or by email."] }
    ]
  },
  cookies: {
    title: "Cookie Policy",
    summary: "The cookies and local storage RYZE uses, what they are for and how to manage your choice.",
    sections: [
      { id: "o-que-sao", title: "1. What cookies are", paragraphs: ["Cookies are small files stored by your browser. We also use the browser's local and session storage, which work in a similar way."] },
      {
        id: "que-usamos",
        title: "2. Cookies we use",
        paragraphs: [
          "RYZE only uses cookies and storage that are strictly necessary for the service and functional ones you request yourself (such as the language). We do not use analytics or advertising cookies."
        ],
        table: { headers: ["Name", "Type", "Purpose", "Category", "Duration"], rows: COOKIE_ROWS_EN }
      },
      {
        id: "consentimento",
        title: "3. Consent",
        paragraphs: [
          "Strictly necessary cookies do not require consent (ePrivacy Directive 2002/58/EC; Portuguese Law 41/2004, art. 5). If we ever use analytics or advertising cookies, we will only enable them with your prior consent, which you can give or refuse in the cookie notice.",
          "You can review your choice at any time with the \"Cookie settings\" button on this page."
        ]
      },
      { id: "gerir", title: "4. Manage in your browser", paragraphs: ["You can also delete or block cookies in your browser settings. If you block strictly necessary cookies, you will not be able to sign in or pay."] }
    ]
  },
  "aviso-legal": {
    title: "Legal Notice",
    summary: "The identity of the company that runs RYZE and the mandatory legal information.",
    sections: [
      {
        id: "identificacao",
        title: "1. Identification",
        list: [`Company: ${E.name}`, `Company number: ${E.taxId}`, `Registered office: ${E.address}`, `Registry: ${E.registry}`, `Share capital: ${E.shareCapital}`, `Email: ${E.email}`]
      },
      { id: "reclamacoes", title: "2. Complaints book", paragraphs: [`You can file a complaint in the Portuguese electronic complaints book: ${COMPLAINTS_BOOK_URL}.`] },
      {
        id: "ral",
        title: "3. Alternative dispute resolution",
        paragraphs: [`For consumer disputes you can use an alternative dispute resolution body such as CNIACC (${CNIACC_URL}). More information on the Consumer Portal: ${CONSUMER_PORTAL_URL}.`]
      },
      { id: "documentos", title: "4. Other documents", paragraphs: ["See also the Terms and Conditions, the Privacy Policy and the Cookie Policy."] }
    ]
  }
};

export const LEGAL_DOCS: Record<"en" | "pt", Record<LegalDocId, LegalDoc>> = { en: EN, pt: PT };
export const LEGAL_ORDER: LegalDocId[] = ["termos", "privacidade", "cookies", "aviso-legal"];
