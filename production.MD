# RYZE — Manual de produção

Manual para preparar, publicar e manter o RYZE online.

- Estado: preparado em 2026-11-27, na versão 2.2.82 (`package.json`) / v2.131.1 (`docs/REGISTRY/update_logs.MD`).
- Nada foi contratado, comprado ou publicado: este documento descreve o que fazer e por que ordem.
- Os preços foram consultados nesta data e estão marcados como **[confirmar]** sempre que a fonte não era oficial ou era ambígua. Confirma-os sempre na página do fornecedor antes de pagar.

Convenções: `ryzecoaching.com` é o domínio pretendido. Se mudar, procura e substitui em `deploy/Caddyfile`, `public/_headers` e nas variáveis de ambiente.

---

## Índice

1. Visão geral
2. Estado atual
3. Contas necessárias
4. Comprar o domínio
5. Contratar o VPS
6. Preparar o servidor
7. Preparar o PostgreSQL
8. Publicar o backend
9. Publicar o frontend
10. DNS e HTTPS
11. CI/CD
12. Backups e recuperação
13. Monitorização
14. Credenciais e variáveis de ambiente
15. Testes antes do lançamento
16. Custos
17. Situações de emergência
18. Preparação comercial e legal
19. Escalabilidade
20. Onde estamos e caminho até ao lançamento (checklists)

---

## 1. Visão geral

### Arquitetura planeada

```
                    Utilizador (browser / PWA)
                              │  HTTPS
                              ▼
                 ┌──────────────────────────────┐
                 │  Cloudflare (DNS + proxy +    │
                 │  CDN + TLS + proteção básica) │
                 └──────┬───────────────┬────────┘
     ryzecoaching.com   │               │  api.ryzecoaching.com
     (site, ficheiros   │               │  (API)
      estáticos)        ▼               ▼
            ┌──────────────────┐   ┌───────────────────────────────────────┐
            │ Cloudflare Pages │   │ VPS OVHcloud (Ubuntu 24.04 LTS)        │
            │  React + Vite    │   │                                       │
            │  (dist/)         │   │  Caddy :443 (único porto público)      │
            └──────────────────┘   │    │  rede Docker privada              │
                                   │    ▼                                  │
                                   │  API Go (container, sem root)          │
                                   │    │                                  │
                                   │    ▼                                  │
                                   │  PostgreSQL 17 (container, SEM porto   │
                                   │  público, volume persistente)          │
                                   │  Volume "backend_data": uploads e      │
                                   │  documentos dos treinadores            │
                                   └──────────────┬────────────────────────┘
                                                  │ backups cifrados (age)
                                                  ▼
                                   ┌───────────────────────────────┐
                                   │ Cloudflare R2 (bucket privado) │
                                   └───────────────────────────────┘

  Serviços externos chamados pela API: Stripe (cartão, MB WAY), PayPal,
  Google (início de sessão), Resend (emails, por SMTP).
  GitHub: código, testes (Actions) e imagens da API (GHCR).
```

### Como as peças comunicam

- **O browser carrega o site** a partir do Cloudflare Pages (`ryzecoaching.com`). Os ficheiros são estáticos: HTML, JS, CSS e ícones.
- **O site chama a API** em `https://api.ryzecoaching.com/api/v1`, com `credentials: "include"`, para enviar o cookie de sessão.
  - O cookie é criado pela API como `HttpOnly`, `Secure` e `SameSite=Lax`.
  - Os dois endereços partilham o domínio registado (`ryzecoaching.com`), por isso o browser trata-os como o mesmo "site" e envia o cookie. **Isto só funciona com os dois no mesmo domínio.** O site num `*.pages.dev` e a API em `ryzecoaching.com` não funcionariam.
  - A API só aceita pedidos de origens listadas em `CORS_ALLOWED_ORIGINS`.
- **Imagens enviadas pelos utilizadores** (capas, avatares, fotos de refeições) são servidas pela API em `https://api.ryzecoaching.com/uploads/...`. O frontend já constrói estes endereços a partir de `VITE_API_BASE_URL`.
- **Pagamentos.** O Stripe e o PayPal notificam a API por webhook (`/api/v1/webhooks/stripe`, `/api/v1/webhooks/paypal`). Uma compra só fica concluída depois dessa confirmação verificada.

### Ambientes

| Ambiente | Site | API | Onde corre |
|---|---|---|---|
| Development | `http://localhost:5173` | `http://localhost:8080` | o teu computador |
| Staging | `https://staging.ryzecoaching.com` | `https://api-staging.ryzecoaching.com` | Pages (branch `staging`) + o mesmo VPS, base de dados própria |
| Production | `https://ryzecoaching.com` | `https://api.ryzecoaching.com` | Pages (branch `main`) + VPS |

Staging e produção partilham o VPS, mas estão separados: cada um tem a sua base de dados, os seus volumes, a sua rede Docker e o seu ficheiro de segredos. Não são precisos três servidores.

---

## 2. Estado atual

### 2.1 O que existe no repositório

| Peça | Estado real (verificado no código) |
|---|---|
| Frontend | React 19 + TypeScript + Vite 8, React Router 5, CSS Modules, Framer Motion, Lucide, Three.js. **O Zustand não está instalado** (a descrição inicial do projeto menciona-o). Typecheck, 189 testes e build de produção passam. |
| PWA | `public/manifest.webmanifest`, `public/sw.js` (cache só de ficheiros estáticos com hash; nunca guarda chamadas à API) e ícones. Só funciona em HTTPS ou `localhost`. |
| Backend | Go 1.26 (imagem Docker com Go 1.27), Gin, GORM, pgx, REST. Arranca, responde em `/api/v1/health` e `/api/v1/version` e para de forma ordenada (`SIGTERM`, 20 s). |
| Base de dados | PostgreSQL. 71 migrações SQL embutidas no binário (`go run . migrate up`). A migração de MariaDB já está feita. |
| Testes backend | 69 pacotes passam: na base de testes e também numa base PostgreSQL nova e vazia, sem `.env` (simulação do CI feita em 2026-11-27). |
| Docker | `main/src/backend/Dockerfile` (imagem distroless, sem shell, utilizador sem privilégios), `main/src/frontend/Dockerfile` (nginx) e `docker-compose.yml` (stack local completa). |
| Servidor sem Docker | `deploy/ryze-backend.service` (systemd), alternativa ao Docker. |
| CI | `.github/workflows/ci.yml` (formatação, build, vet, testes, frontend e E2E). |
| Backups locais | `scripts/backup_db.sh`, `scripts/restore_db.sh` (cópia local; cifra opcional com GPG). |
| Jobs em background | Dentro do processo da API: envio de emails, comprovativos, ofertas do Premium Nível 2, limpeza de notificações, anonimização de contas apagadas há mais de 30 dias e lembretes de treino. Todos bloqueiam linhas na base, por isso várias instâncias não se atropelam. |
| SSE / tempo real | **Não existe.** O único stream (chat de suporte) foi removido na v2.131.0. As notificações são lidas por pedidos normais. A descrição inicial menciona SSE e um "sistema de suporte": já não se aplicam. |
| Logs | Texto para stdout/stderr (`log/slog`). Em Docker ficam em `docker logs`. |

### 2.2 Respostas às perguntas de estado

1. **Versão:** 2.2.82 (`package.json`), changelog v2.131.1, última migração 000071.
2. **Frontend:** funcional e testado localmente. Para produção falta só a configuração (`VITE_API_BASE_URL`) e o alojamento. Tem textos legais com campos por preencher (identidade da empresa).
3. **Backend:** funcional e testado. Falta configurar credenciais reais (pagamentos, email, Google).
4. **PostgreSQL:** esquema e migrações prontos. Falta o servidor, os backups automáticos externos e o teste de restauro em produção.
5. **Funciona localmente?** Sim, com `./start_test_mode.sh` (PostgreSQL local + API + Vite). Com pagamentos, email e Google por configurar, essas funções ficam desligadas ou em modo de teste.
6. **Dependências de configurações locais:** os valores por omissão apontam para desenvolvimento (`CORS_ALLOWED_ORIGINS` = `http://localhost:5173`, `DB_HOST` = `127.0.0.1`). Em produção têm de ser definidos; o `deploy/compose.app.yml` e o `app.env` tratam disso.
7. **Endpoints com `localhost`:** só como valor por omissão de desenvolvimento.
   - `main/src/utils/http_client.ts`: sem `VITE_API_BASE_URL`, usa `<host>:8080`. O build de produção define a variável.
   - `config/env.go`: CORS por omissão.
   - `email/smtp.go` e `contact/smtp_notifier.go` usam `Hello("localhost")` no SMTP. É inofensivo e aceite pelo Resend.
8. **Caminhos absolutos do computador de desenvolvimento:** nenhum no código. O `vite.config.ts` usa caminhos relativos.
9. **Funcionalidades dependentes de serviços por configurar:**
   - Stripe: cartão e MB WAY.
   - PayPal.
   - Resend: emails (verificação de conta, comprovativos, lembretes).
   - Google: início de sessão com Google.
   - Formulário de contacto: SMTP de contacto.
   - Sem estes serviços a aplicação arranca. As funções correspondentes ficam indisponíveis ou os emails ficam registados como "skipped", sem dizer que foram enviados.
10. **Só com Docker local?** Não. A aplicação corre sem Docker (`start_test_mode.sh`) e com Docker (`docker-compose.yml`).
11. **Problemas que impediam a publicação** (corrigidos nesta revisão):
    - Na imagem Docker, os uploads iam para `/uploads`, onde o utilizador sem privilégios não pode escrever, e fora do volume persistente. Corrigido com `UPLOADS_DIR=/var/lib/ryze/uploads` no Dockerfile e nos Compose.
    - O nginx da stack Docker local não reencaminhava `/uploads/` para a API, por isso as imagens não apareciam. Corrigido.
    - Um teste de ordenação por nome dependia da collation da base e falharia numa base nova. Corrigido.

    **Bloqueios que continuam abertos** (ver §18): a licença dos media dos exercícios, a identidade legal nos documentos, a faturação certificada e a titularidade das contas de pagamento.
12. **VPS:** sim. A arquitetura é um processo Go sem estado + PostgreSQL + ficheiros num volume, e cabe num VPS pequeno.

### 2.3 Prontidão por área

Legenda: **PRONTO** · **PARCIAL** (parcialmente pronto) · **NÃO PREP.** (não preparado) · **CRED.** (bloqueado por credenciais) · **EXT.** (bloqueado por serviço externo) · **DECISÃO** (bloqueado por decisão comercial ou legal).

| Área | Estado | Notas |
|---|---|---|
| Build de produção | PARCIAL | O frontend constrói. A imagem da API não foi construída nesta máquina (sem permissão para usar o Docker); o primeiro build acontece no GitHub Actions. |
| Configuração de ambiente | PARCIAL | Modelos prontos (`.env.example`, `deploy/`). Faltam os ficheiros reais de cada ambiente no servidor. |
| Segurança (aplicação) | PRONTO | Cookies `HttpOnly`/`Secure`/`SameSite`, CSRF, CORS explícito, rate limiting, CSP, uploads re-codificados, segredos fora do Git. |
| Segurança (servidor) | NÃO PREP. | Passos em §6. |
| Autenticação | CRED. | O código está pronto. Faltam `JWT_SECRET` e contas de admin novas por ambiente, e o Google para o login com Google. |
| PostgreSQL | PARCIAL | Esquema e migrações prontos; o servidor não existe. |
| Migrações | PRONTO | Aplicadas pelo `deploy/deploy.sh`, sempre depois de um backup. **Nunca usar `migrate down`**: reverte todas as migrações. |
| Pagamentos | CRED. + DECISÃO | Precisam de contas Stripe/PayPal de um titular válido (§18). |
| Google OAuth | CRED. | Projeto no Google Cloud e ecrã de consentimento. |
| Resend (email) | CRED. + EXT. | Conta, domínio verificado (registos DNS) e chave. |
| Uploads | PARCIAL | Volume persistente local corrigido; backup externo pronto (§12). O armazenamento S3/R2 direto não está implementado (§19). |
| Notificações | PRONTO | Na app; o email depende do Resend. |
| SSE / reconexões | Não se aplica | Não existe SSE desde a v2.131.0. |
| PWA | PRONTO | Precisa de HTTPS (Cloudflare). |
| Logs | PARCIAL | Para stdout; falta a rotação do Docker (§6.6). |
| Backups | PARCIAL | Scripts prontos (`deploy/backup.sh`, `deploy/restore.sh`), nunca corridos num servidor real. Falta o bucket R2 e a chave `age`. |
| Monitorização | NÃO PREP. | Proposta em §13. |
| HTTPS | NÃO PREP. | Depende do domínio e do Cloudflare; configuração preparada (`deploy/Caddyfile`). |
| CORS | PRONTO | `CORS_ALLOWED_ORIGINS=https://ryzecoaching.com`. |
| Rate limiting | PRONTO | Em memória, por instância. O IP real chega pelo Caddy, que só confia nos IPs da Cloudflare. |
| Gestão de sessões | PRONTO | JWT em cookie (`JWT_ACCESS_TOKEN_TTL`, 15 min por omissão) e `session_version` para invalidar sessões. |
| Recuperação de erros | PARCIAL | Paragem ordenada, `restart: unless-stopped`, rollback automático da imagem no deploy. Faltam alertas. |
| Media dos exercícios | DECISÃO | Desligados (`EXERCISE_MEDIA_ENABLED=false`) até haver licença comercial do conjunto "© Gym visual" (§18). |

### 2.4 Resumo honesto

- **Pode ir já para staging,** depois de contratar o domínio e o VPS: site, contas, planos grátis, treinos, progresso e admin. Pagamentos em modo de teste (Stripe test/PayPal sandbox) e emails do Resend.
- **Precisa só de configuração:** domínio, DNS, HTTPS, servidor, ficheiros de segredos, Pages e backups.
- **Precisa de credenciais:** Stripe, PayPal, Resend, Google, Cloudflare (token e R2).
- **Depende de decisões comerciais ou legais:** receber pagamentos reais, emitir faturas, dados da empresa nos textos legais e licença dos media dos exercícios.
- **Precisa de verificação antes de abrir ao público:** teste de restauro, teste de carga, testes de pagamento ponta a ponta, revisão jurídica.

---

## 3. Contas necessárias

> **Não crio contas por ti.** Algumas exigem um titular maior de idade e, mais tarde, a empresa (ver §18). Abre-as em nome de quem vai responder pelo serviço e pagar as faturas.

| Conta | Para quê | Link oficial | Plano | O que vais tirar de lá | Faturação / elegibilidade |
|---|---|---|---|---|---|
| Cloudflare | Domínio, DNS, proxy, TLS, Pages, R2 | https://dash.cloudflare.com/sign-up | Free (+ registo do domínio, pago) | API token (Pages), Account ID, chaves R2, certificado de origem | Email verificado e método de pagamento principal para o domínio. Os termos exigem capacidade para contratar **[confirmar idade mínima nos termos]**. |
| OVHcloud | VPS | https://www.ovhcloud.com/pt/vps/ | VPS-2 (recomendado) | IP do servidor, acesso SSH | Pagamento e identificação. Contrato com quem tenha capacidade legal **[confirmar]**. |
| GitHub | Código, Actions, imagens (GHCR) | https://github.com | Free (o repositório é **público**) | Secrets e variáveis dos Environments | Já existe. |
| Resend | Emails da aplicação | https://resend.com | Free (3 000/mês, 100/dia) | Chave API (usada como palavra-passe SMTP) | Domínio verificado por DNS. |
| Google Cloud | Início de sessão com Google | https://console.cloud.google.com | Gratuito | Client ID, Client Secret | Ecrã de consentimento OAuth. |
| Stripe | Cartão e MB WAY | https://dashboard.stripe.com/register | Por transação | Secret key, publishable key, webhook secret | **Verificação de identidade (KYC) do titular e dados bancários.** Exige o titular adulto ou a empresa (§18). |
| PayPal Business | PayPal | https://www.paypal.com/pt/business | Por transação | Client ID, Secret, Webhook ID | Conta Business; titular maior de idade (§18). |
| Monitorização (opcional) | Avisar se o site ou os backups falharem | https://uptimerobot.com, https://healthchecks.io | Free | URL de ping | Plano gratuito **[confirmar limites]**. |

---

## 4. Comprar o domínio

### 4.1 Conceitos rápidos

- **Domínio:** o nome (`ryzecoaching.com`). Paga-se por ano a um registrar e fica teu enquanto o renovares.
- **DNS:** a "lista telefónica" que diz para onde aponta cada nome (por exemplo, `api.ryzecoaching.com` → IP do VPS).
- **Subdomínio:** um nome por baixo do teu domínio (`api.`, `staging.`). Não se paga à parte.
- **HTTPS:** a ligação cifrada (cadeado). A Cloudflare emite o certificado para os visitantes. Entre a Cloudflare e o VPS usa-se um "certificado de origem" grátis da Cloudflare (§10).

### 4.2 Disponibilidade (verificada)

- Em **2026-11-27**, o registo oficial dos `.com` (RDAP da Verisign) respondeu **404 — não registado** para `ryzecoaching.com`.
- Isto **não é uma reserva**: até o comprares, outra pessoa pode registá-lo.
- O registrar pode ainda marcá-lo como "premium" (mais caro). Confirma o preço no ecrã de compra.

### 4.3 Preço (Cloudflare Registrar)

- A Cloudflare vende ao custo, sem margem: paga-se o preço do registo mais a taxa da ICANN. A renovação faz-se ao mesmo preço de tabela.
- `.com`: cerca de **11,17 USD/ano**, depois de o preço grossista da Verisign subir em 2026-11-01 **[confirmar no ecrã de compra; o IVA pode acrescer]**.
- Não há reembolsos: o registo e as renovações são finais.
- Renovação automática ligada por omissão (cerca de 30 dias antes do fim).
- **Métodos de pagamento:** a documentação só diz que é usado o "método de pagamento principal" da conta; não lista quais **[confirmar no painel de faturação]**.

### 4.4 Requisitos e condições

- Conta Cloudflare com o email verificado.
- Os domínios registados na Cloudflare **usam obrigatoriamente o DNS da Cloudflare** (é o que queremos).
- O email do titular tem de ser verificado (regra da ICANN). Se não for, o domínio é suspenso.
- **Transferências:** depois do registo há um bloqueio de 60 dias (regra da ICANN). A seguir podes desbloquear, obter o código de autorização e mudar de registrar.
- **Privacidade:** os dados do titular ficam ocultos no WHOIS por omissão; só o país e a região ficam visíveis.
- **DNSSEC:** ativa-se com um clique e é gratuito.

### 4.5 Alternativas

- Outros registrars (Porkbun, Namecheap, OVHcloud) cobram preços parecidos no primeiro ano e quase sempre renovam mais caro **[confirmar se for considerado]**.
- Não há razão técnica para mudar: a Cloudflare dá registo, DNS e Pages no mesmo sítio.
- Se o `.com` deixar de estar disponível, **não compres nenhuma alternativa sem decidir**. Hipóteses para escolheres:
  - `ryzecoaching.pt` (requer o registo .PT);
  - `ryze.coach`;
  - `getryze.com`;
  - `ryzeapp.com`.

### 4.6 Passo a passo (quando decidires comprar)

1. Cria a conta Cloudflare e ativa a **autenticação de dois fatores** (Perfil → Autenticação).
2. Painel → **Domain Registration → Register Domains** → escreve `ryzecoaching.com`.
3. Confirma o preço, a renovação e o IVA. Preenche os contactos do titular (dados reais; ver §18 sobre quem é o titular).
4. Escolhe 1 ano, mantém a renovação automática ligada e conclui a compra.
5. Verifica o email da ICANN que chega à caixa do titular.
6. Em **DNS → Settings**, ativa o **DNSSEC**.
7. Não cries ainda registos DNS: espera pelo IP do VPS (§10).

---

## 5. Contratar o VPS

### 5.1 Conceitos

- **VPS:** uma máquina virtual com recursos reservados (CPU, RAM, disco), a correr no centro de dados do fornecedor. Instalas e geres tudo.
- **CPU / vCores:** capacidade de cálculo. A API Go usa pouca; conta sobretudo em picos de pedidos e na compressão dos backups.
- **RAM:** memória de trabalho. Se acabar, o Linux mata processos (o PostgreSQL ou a API) ou fica muito lento.
- **NVMe:** disco SSD rápido. Guarda a base de dados, os uploads, as imagens Docker e os backups locais.
- **Sem recursos:**
  - RAM esgotada: processos terminados e serviço em baixo.
  - Disco cheio: o PostgreSQL para de escrever e a aplicação falha.
  - CPU esgotada: lentidão.

  A §13 diz como vigiar cada um.

### 5.2 Consumo medido (2026-11-27, ambiente local)

- **API em execução:** cerca de 36 MB de RAM.
- **PostgreSQL:** cerca de 150 MB no total (todos os processos).
- **Dados:** base comprimida cerca de 0,5 MB; uploads cerca de 2 MB.
- **Na estimativa** acrescento o Caddy (~30 MB), o Docker (~100 MB), o sistema (~400 MB) e um segundo par API + PostgreSQL para staging.
- **Total estimado:** menos de 1,5 GB de RAM, com folga.

> Isto **não é um teste de carga.** Antes de abrir ao público, corre o teste em §15 no staging para confirmar o comportamento com dezenas de pedidos simultâneos.

### 5.3 Opções (preços da página oficial da OVH em 2026-11-27)

| Plano | vCores | RAM | NVMe | Rede | Preço/mês sem IVA | com IVA (PT) |
|---|---|---|---|---|---|---|
| OVH VPS-1 | 2 | 4 GB | 40 GB | 500 Mbps | 3,81 € | 4,69 € |
| **OVH VPS-2** | 4 | 8 GB | 75 GB | 1 Gbps | 7,21 € | **8,87 €** |
| OVH VPS-3 | 6 | 12 GB | 100 GB | 2 Gbps | 10,40 € | 12,79 € |
| Hetzner CX23 | 2 | 4 GB | 40 GB | — | 5,49 € + 0,50 € IPv4 | **[confirmar IVA e disponibilidade]** |
| Hetzner CX33 | 4 | 8 GB | 80 GB | — | 8,49 € + 0,50 € IPv4 | **[confirmar]** |

Notas:

- A OVH indica **tráfego ilimitado** e **backup automático diário incluído**.
- O blog da OVH anunciou outros preços para a família "VPS 2026". A página de compra é que manda: **confirma no carrinho**.
- Há promoções com meses grátis contra compromisso de 6 ou 12 meses **[confirmar condições]**.
- Várias fontes dão a Hetzner como esgotada em alguns planos (novembro de 2026).

**Recomendação: OVH VPS-2 (8,87 €/mês).**

- 8 GB chegam para produção e staging lado a lado, com espaço para crescer.
- Os 75 GB chegam para anos de dados ao ritmo atual.
- **Mais económico:** VPS-1 (4,69 €). Chega para produção e um staging leve pelos números medidos, mas tem menos margem para picos e para a compressão de backups.
- **Para crescimento:** VPS-3 (12,79 €).
- **Região:** um centro de dados na UE (por exemplo França ou Alemanha), por causa da latência para Portugal e do RGPD.
- **Sistema:** Ubuntu 24.04 LTS.

### 5.4 Aumentar mais tarde

- A OVH permite subir de VPS-1 para VPS-2 ou VPS-3 no painel **[confirmar se o seu modelo atual permite upgrade direto]**.
- Os dados mantêm-se e o servidor reinicia.
- Conta com alguns minutos de indisponibilidade; faz um backup antes.
- Descer de plano costuma não ser possível.
- **Separar o PostgreSQL** só compensa quando:
  - a base precisar de mais RAM do que o servidor pode dar;
  - ou houver várias instâncias da API.

  Nessa altura: um segundo VPS na rede privada, ou um PostgreSQL gerido. Mais custo; ver §19.

### 5.5 Contratar (quando decidires)

1. Escolhe o VPS-2, região na UE, Ubuntu 24.04, faturação mensal.
2. Na encomenda, adiciona a tua **chave SSH pública** se a OVH o permitir (§6.1). Assim o servidor nem aceita palavra-passe.
3. Paga. Recebes por email o IP e o utilizador inicial (normalmente `ubuntu`).
4. Guarda o IP. Vais precisar dele no DNS (§10) e no GitHub (§11).

---

## 6. Preparar o servidor

> Os comandos são para correr **no VPS**, salvo indicação. Cada um tem uma explicação. Não corras nada que não percebas; pergunta primeiro.

### 6.1 Chaves SSH (no teu computador)

```bash
ssh-keygen -t ed25519 -C "ryze-admin" -f ~/.ssh/ryze_admin     # chave para ti (com frase-passe)
ssh-keygen -t ed25519 -C "ryze-deploy" -f ~/.ssh/ryze_deploy -N ""  # chave só para o GitHub Actions
```

- As chaves `.pub` são públicas e vão para o servidor.
- Os ficheiros sem `.pub` são privados e nunca saem do teu computador, exceto a chave de deploy, que vai para um secret do GitHub (§11).

### 6.2 Primeiro acesso, atualizações e utilizadores

```bash
ssh -i ~/.ssh/ryze_admin ubuntu@<IP_DO_VPS>     # entrar
sudo apt update && sudo apt full-upgrade -y      # atualizar o sistema
sudo apt install -y unattended-upgrades fail2ban age rclone curl
sudo dpkg-reconfigure -plow unattended-upgrades  # instalar atualizações de segurança automaticamente
sudo adduser --disabled-password --gecos "" deploy   # utilizador que o GitHub usa para publicar
```

- `fail2ban` bloqueia tentativas repetidas de SSH.
- `age` cifra os backups.
- `rclone` envia os backups para o R2.

Chave do deploy (cola o conteúdo de `ryze_deploy.pub`):

```bash
sudo install -d -m 700 -o deploy -g deploy /home/deploy/.ssh
echo 'restrict ssh-ed25519 AAAA...conteúdo-de-ryze_deploy.pub... ryze-deploy' | sudo tee /home/deploy/.ssh/authorized_keys
sudo chown deploy:deploy /home/deploy/.ssh/authorized_keys && sudo chmod 600 /home/deploy/.ssh/authorized_keys
```

`restrict` desliga reencaminhamentos e terminal interativo para esta chave.

### 6.3 SSH só com chave

```bash
sudo tee /etc/ssh/sshd_config.d/90-ryze.conf <<'EOF'
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
EOF
sudo systemctl restart ssh
```

> Antes de fechar a sessão, abre **outra** janela e confirma que ainda entras (`ssh -i ~/.ssh/ryze_admin ubuntu@<IP>`). Se algo correr mal, a sessão antiga ainda está aberta para corrigir.

### 6.4 Firewall

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status verbose
```

- O PostgreSQL **não** é aberto: não tem porto publicado no Compose.
- **Atenção:** o Docker abre os portos que publica (80 e 443 do Caddy) por fora do UFW. Isso é aceitável aqui porque o Caddy só responde aos nomes configurados e com o certificado de origem.
- **Opcional, mais restrito:** aceitar 443 só dos IPs da Cloudflare, com a firewall de rede da OVH ou regras na cadeia `DOCKER-USER`. Faz isto só depois de tudo funcionar.

### 6.5 Docker

```bash
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo $VERSION_CODENAME) stable" | sudo tee /etc/apt/sources.list.d/docker.list
sudo apt update && sudo apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
sudo usermod -aG docker deploy
```

> **Impacto de segurança:** quem está no grupo `docker` consegue, na prática, obter privilégios de administrador. Aceita-se aqui porque a chave do `deploy` só serve para o GitHub publicar. **Não dês essa chave a mais ninguém.** O `ubuntu` (tu) faz a manutenção com `sudo`.

### 6.6 Rotação de logs do Docker

Sem isto, os logs crescem até encher o disco.

```bash
sudo tee /etc/docker/daemon.json <<'EOF'
{ "log-driver": "json-file", "log-opts": { "max-size": "20m", "max-file": "5" } }
EOF
sudo systemctl restart docker
```

### 6.7 Pastas e permissões

```bash
sudo install -d -m 755 -o deploy -g deploy /opt/ryze /opt/ryze/deploy
sudo install -d -m 700 -o deploy -g deploy /opt/ryze/production /opt/ryze/staging /opt/ryze/backups /opt/ryze/edge /opt/ryze/edge/certs
```

Copia os ficheiros de `deploy/` do repositório para `/opt/ryze/deploy/` (a partir do teu computador):

```bash
scp -i ~/.ssh/ryze_admin deploy/compose.app.yml deploy/compose.edge.yml deploy/deploy.sh deploy/backup.sh deploy/restore.sh ubuntu@<IP>:/tmp/
ssh -i ~/.ssh/ryze_admin ubuntu@<IP> 'sudo mv /tmp/compose.*.yml /tmp/*.sh /opt/ryze/deploy/ && sudo chown deploy:deploy /opt/ryze/deploy/* && sudo chmod 755 /opt/ryze/deploy/*.sh'
scp -i ~/.ssh/ryze_admin deploy/Caddyfile ubuntu@<IP>:/tmp/ && ssh -i ~/.ssh/ryze_admin ubuntu@<IP> 'sudo mv /tmp/Caddyfile /opt/ryze/edge/ && sudo chown deploy:deploy /opt/ryze/edge/Caddyfile'
echo "ghcr.io/edgar-git/ryze-api" | ssh -i ~/.ssh/ryze_admin ubuntu@<IP> 'sudo tee /opt/ryze/image-repository'
```

Estrutura final:

```
/opt/ryze/
├── image-repository          # ghcr.io/<owner>/ryze-api
├── backup.env                # chaves do backup (600)
├── deploy/                   # compose.app.yml, compose.edge.yml, *.sh (cópia do repositório)
├── production/  app.env db.env release.env deployments.log   (700, ficheiros 600)
├── staging/     app.env db.env release.env deployments.log
├── edge/        Caddyfile  certs/origin.pem certs/origin.key
└── backups/     cópias locais dos últimos 3 dias (cifradas)
```

> Se mudares algo em `deploy/` no repositório, volta a copiar para o servidor: estes ficheiros não são atualizados pelo CI, de propósito, para o servidor não executar código novo sem revisão.

---

## 7. Preparar o PostgreSQL

- **Onde corre:** num container `postgres:17`, um por ambiente, com volume persistente (`ryze-production_db_data`).
- **Acesso:** só pela rede Docker do ambiente; sem porto público.
- **Credenciais:** em `/opt/ryze/<ambiente>/db.env` (600):

```bash
sudo -u deploy bash -c 'umask 077; cat > /opt/ryze/production/db.env' <<EOF
POSTGRES_DB=ryze
POSTGRES_USER=ryze_app
POSTGRES_PASSWORD=$(openssl rand -base64 36 | tr -d '/+=')
EOF
```

- Os mesmos três valores vão para o `app.env` como `DB_NAME`, `DB_USER` e `DB_PASSWORD` (§8.1).
- Usa valores **diferentes** em staging.
- O utilizador da aplicação é dono da base, o que lhe permite criar a extensão `citext` de que as migrações precisam.

**Migrações:**

- O `deploy.sh` aplica-as sempre, com a imagem nova, **depois de um backup** e antes de trocar a API.
- O log regista a versão do esquema antes e depois.
- Para ver o estado à mão:

```bash
cd /opt/ryze/production
docker compose --project-directory . --env-file release.env -f /opt/ryze/deploy/compose.app.yml run --rm --no-deps backend migrate version
```

- **Regra de desenvolvimento:** cada migração tem de ser compatível com a versão anterior da API (adicionar colunas e tabelas; renomear ou apagar só numa versão seguinte). O rollback automático volta à imagem anterior da API, mas **não desfaz migrações**.
- **Nunca** correr `migrate down` (reverte TODAS as migrações).

Backups e restauro: §12.

---

## 8. Publicar o backend

### 8.1 Ficheiro de segredos de cada ambiente

Em `/opt/ryze/production/app.env` (600, dono `deploy`). Usa a lista de §14 e o `.env.example` como modelo.

O mínimo para arrancar:

```bash
sudo -u deploy bash -c 'umask 077; nano /opt/ryze/production/app.env'
```

```
DB_NAME=ryze
DB_USER=ryze_app
DB_PASSWORD=<igual ao db.env>
JWT_SECRET=<openssl rand -hex 48>
ADMIN_1_USERNAME=<nome>
ADMIN_1_PASSWORD=<openssl rand -base64 24>
ADMIN_1_ACCESS_CODE=<openssl rand -hex 8>
ADMIN_2_USERNAME=<outro nome>
ADMIN_2_PASSWORD=<...>
ADMIN_2_ACCESS_CODE=<...>
CORS_ALLOWED_ORIGINS=https://ryzecoaching.com
PUBLIC_APP_URL=https://ryzecoaching.com
TEST_MODE_ENABLED=false
EXERCISE_MEDIA_ENABLED=false
```

Os valores de staging são outros (`https://staging.ryzecoaching.com`) e podem ter `TEST_MODE_ENABLED=true`, se quiseres o Modo de Teste em staging.

### 8.2 Primeiro arranque

O primeiro deploy pode ser feito pelo GitHub (§11) ou à mão:

```bash
sudo -u deploy /opt/ryze/deploy/deploy.sh production <tag-da-imagem>
sudo -u deploy /opt/ryze/deploy/deploy.sh staging <tag-da-imagem>
cd /opt/ryze/edge && sudo -u deploy docker compose -f /opt/ryze/deploy/compose.edge.yml up -d
```

- A tag aparece no GitHub em **Packages → ryze-api** (formato `2.2.82-<commit>`).
- O que o `deploy.sh` faz:
  1. puxa a imagem;
  2. arranca a base;
  3. faz um backup;
  4. aplica as migrações;
  5. troca a API;
  6. testa `/api/v1/health`.
- Se algo falhar, volta à imagem anterior e regista tudo em `deployments.log`.
- **Indisponibilidade num deploy:** alguns segundos enquanto o container da API é recriado.
- **Acesso à imagem.** O repositório é público, mas um pacote novo no GHCR pode nascer privado **[confirmar no primeiro build: GitHub → Packages → ryze-api → Package settings]**.
  - Se estiver privado, faz uma vez, como `deploy`, `docker login ghcr.io` com um token do GitHub (classic) só com `read:packages`.
  - Em alternativa, torna o pacote público. A imagem não contém segredos, mas fica visível para todos, tal como o código.

### 8.3 Verificar, ver logs, voltar atrás

```bash
curl -s https://api.ryzecoaching.com/api/v1/health      # {"status":"ok"} = API e base de dados bem
curl -s https://api.ryzecoaching.com/api/v1/version     # versão instalada
cat /opt/ryze/production/release.env                    # imagem atual e anterior
tail -n 50 /opt/ryze/production/deployments.log         # histórico
cd /opt/ryze/production && docker compose --project-directory . --env-file release.env -f /opt/ryze/deploy/compose.app.yml logs --tail 200 backend
```

**Voltar atrás** (rollback manual para uma tag anterior):

```bash
sudo -u deploy /opt/ryze/deploy/deploy.sh production <tag-anterior>
```

Também faz um backup e aplica `migrate up`. Uma imagem antiga ignora migrações mais recentes que já existam, desde que sejam compatíveis (§7).

---

## 9. Publicar o frontend (Cloudflare Pages)

**Compatibilidade verificada:**

- O build (`npm run build`) gera ficheiros estáticos em `dist/`.
- O React Router funciona: o Pages trata projetos **sem `404.html` na raiz** como SPA e devolve o `index.html` para caminhos desconhecidos. O RYZE não tem `404.html`.
- `public/_headers` aplica os cabeçalhos de segurança (incluindo a CSP com o Stripe e a API) e o cache dos ficheiros com hash.
- O plano gratuito permite 500 builds por mês, um de cada vez, 100 domínios por projeto e até 20 000 ficheiros por site. O RYZE gera muito menos.

**Como publicar.** Os deploys são feitos pelo GitHub Actions (§11), com upload direto (`wrangler pages deploy`), e **não** pela ligação Git do Pages. Assim o site novo só sai **depois** de a API nova estar no ar e saudável.

Configuração única:

1. Cloudflare → **Workers & Pages → Create → Pages → Upload assets** (ou "Direct Upload"). Nome do projeto: `ryze`. Ramo de produção: `main`.
2. **Custom domains** do projeto → adicionar `ryzecoaching.com` e `www.ryzecoaching.com`. Com o DNS na Cloudflare, os registos são criados automaticamente.
3. Staging:
   - o ramo `staging` gera o endereço `staging.ryze.pages.dev`;
   - cria o registo `CNAME staging → staging.ryze.pages.dev` (proxied);
   - para esconder o staging do público, usa **Cloudflare Access** (Zero Trust, plano gratuito **[confirmar limites]**) com uma regra só para os vossos emails.
4. Redirecionar `www` para o domínio principal: **Rules → Redirect Rules** → "Redirect from WWW to root" (modelo da Cloudflare, 301).

**Variável do build** (Environment do GitHub, §11):

- produção: `VITE_API_BASE_URL=https://api.ryzecoaching.com/api/v1`;
- staging: `VITE_API_BASE_URL=https://api-staging.ryzecoaching.com/api/v1`.

O valor fica escrito no bundle. Mudá-lo obriga a um novo deploy.

**HTTPS:** automático no Pages com domínio próprio.

---

## 10. DNS e HTTPS

### 10.1 Registos a criar (depois de teres o IP do VPS)

| Nome | Tipo | Destino | Proxy | Para quê |
|---|---|---|---|---|
| `ryzecoaching.com` | criado pelo Pages | projeto `ryze` | Sim | site |
| `www` | criado pelo Pages | projeto `ryze` | Sim | redireciona para o domínio principal |
| `api` | A | `<IP_DO_VPS>` | **Sim** (laranja) | API de produção |
| `api-staging` | A | `<IP_DO_VPS>` | Sim | API de staging |
| `staging` | CNAME | `staging.ryze.pages.dev` | Sim | site de staging |
| (Resend) | TXT/MX/CNAME | os que o Resend indicar | **Não** (cinzento) | SPF/DKIM para o email |
| `_dmarc` | TXT | `v=DMARC1; p=none; rua=mailto:<o teu email>` | Não | política de email (começar com `p=none`) |

Se o VPS tiver IPv6, cria também registos `AAAA`. Não cries outros subdomínios.

### 10.2 HTTPS entre a Cloudflare e o VPS

1. Cloudflare → **SSL/TLS → Overview** → modo **Full (strict)**.
2. **SSL/TLS → Origin Server → Create Certificate**: chave RSA, nomes `ryzecoaching.com, *.ryzecoaching.com`, validade 15 anos.
3. Copia o certificado para `/opt/ryze/edge/certs/origin.pem` e a chave privada para `/opt/ryze/edge/certs/origin.key` (`chmod 600`). A chave só aparece uma vez.
4. Em **SSL/TLS → Edge Certificates**, liga "Always Use HTTPS" e define "Minimum TLS Version" = 1.2.

O certificado de origem só é aceite pela Cloudflare. Por isso a API **tem** de passar pelo proxy laranja.

### 10.3 Cuidados para não partir funcionalidades

- **Webhooks** (Stripe/PayPal em `/api/v1/webhooks/*`): não ativar "Bot Fight Mode" nem desafios (CAPTCHA) para `api.` e não adicionar regras WAF que bloqueiem pedidos `POST` sem browser. Se ativares regras, cria antes uma exceção para `/api/v1/webhooks/`.
- **Cache:** a Cloudflare não guarda respostas da API, que são dinâmicas e com cookies. Não crias "Cache Rules" para `api.`.
- **Rate limiting:** já existe na API. Não precisas do pago da Cloudflare para começar.
- **Ligações longas:** já não há SSE. Os pedidos normais ficam muito abaixo do limite de 100 s do proxy.
- **CORS:** a API devolve os cabeçalhos; a Cloudflare não os altera.

---

## 11. CI/CD

### 11.1 Como fica o dia a dia

1. Trabalhas no teu computador.
2. `git push` para o ramo `staging` → o GitHub corre os testes, constrói a imagem, publica a API de staging e depois o site de staging.
3. Testas em `https://staging.ryzecoaching.com`.
4. Juntas `staging` em `main` e fazes `git push` → o mesmo processo para produção, mas **espera pela tua aprovação** no GitHub (Actions → o workflow → "Review deployments" → Approve).
5. A versão nova fica no ar; `GET /api/v1/version` mostra-a.

Se um teste falhar, nada é publicado. Se a API nova não responder, o servidor volta sozinho à anterior e o site não é publicado.

### 11.2 Ficheiros

- `.github/workflows/ci.yml`: testes em cada push e pull request. Agora inclui as migrações e a suite completa num PostgreSQL 17 temporário. É também chamado pelo deploy.
- `.github/workflows/deploy.yml`, por ordem:
  1. testes;
  2. imagem `ghcr.io/<owner>/ryze-api:<versão>-<commit>`;
  3. SSH para o VPS com `deploy.sh`;
  4. health check público;
  5. Cloudflare Pages.
- `deploy/deploy.sh`, `backup.sh`, `restore.sh`: correm no VPS.

### 11.3 Configuração única no GitHub

**Settings → Environments.**

1. Cria o Environment `production` com **Required reviewers** (tu) e "Deployment branches" = `main`.
2. Cria o Environment `staging` com "Deployment branches" = `staging`.

**Secrets** (em cada Environment):

| Nome | Valor |
|---|---|
| `VPS_HOST` | IP do VPS |
| `VPS_SSH_KEY` | conteúdo do ficheiro privado `~/.ssh/ryze_deploy` |
| `VPS_SSH_KNOWN_HOSTS` | resultado de `ssh-keyscan -t ed25519 <IP>` (confirma a impressão digital com `ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub` no servidor) |
| `CLOUDFLARE_API_TOKEN` | token com a permissão "Cloudflare Pages: Edit" (só isso) |
| `CLOUDFLARE_ACCOUNT_ID` | ID da conta (painel, barra lateral) |

**Variables** (em cada Environment):

| Nome | production | staging |
|---|---|---|
| `VITE_API_BASE_URL` | `https://api.ryzecoaching.com/api/v1` | `https://api-staging.ryzecoaching.com/api/v1` |
| `CLOUDFLARE_PAGES_PROJECT` | `ryze` | `ryze` |
| `VPS_USER` | `deploy` | `deploy` |

- Os secrets nunca aparecem nos logs: o GitHub mascara-os e os scripts não os imprimem.
- O token do GitHub para publicar a imagem é automático (`GITHUB_TOKEN`), com permissão só para pacotes.

### 11.4 Notas

- O ramo `staging` ainda não existe. Cria-o quando o staging estiver pronto (`git switch -c staging && git push -u origin staging`).
- O teste E2E (Playwright) corre no CI. Se se tornar lento ou instável, pode passar a correr só nos pull requests.
- **Não foi possível correr este workflow localmente.** A primeira execução no GitHub é o teste real; vê os logs com atenção.

---

## 12. Backups e recuperação

### 12.1 Estratégia (proposta)

| O quê | Como | Frequência | Destino | Retenção |
|---|---|---|---|---|
| Base de dados | `pg_dump` dentro do container + gzip + `age` | diária às 03:30 UTC, e antes de cada deploy | R2 `ryze-backups/<env>/daily`, `pre-deploy` | 14 dias |
| Uploads e documentos | `tar` do volume + `age` | igual | igual | 14 dias |
| Semanal | cópia de domingo | semanal | `weekly/` | 8 semanas |
| Mensal | cópia do dia 1 | mensal | `monthly/` | 12 meses |
| Backup da OVH | incluído no VPS | diário | OVH | 1 dia (complementar, não único) |

- **Tamanho estimado:** menos de 5 MB por cópia hoje, por isso o total fica muito abaixo dos 10 GB grátis do R2.
- **Custo previsto:** 0 €.
- **Cifra:**
  - Os backups são cifrados com a **chave pública** `age`, que fica no servidor.
  - A **chave privada** fica fora do servidor: num gestor de palavras-passe e numa cópia em papel ou USB guardada em sítio seguro.
  - Quem roubar o bucket ou o servidor não consegue ler os backups. **Se perderes a chave privada, perdes os backups.**

### 12.2 RPO e RTO (propostas para aprovares)

- **RPO** (perda máxima de dados): **24 horas** (backup diário), ou menos quando o último deploy é mais recente.
- **RTO** (tempo para voltar ao ar com um servidor novo): **4 horas**, seguindo §12.5.

### 12.3 Configuração única

1. Cloudflare → **R2** → criar o bucket `ryze-backups` (privado, localização na UE se disponível).
2. **R2 → Manage API tokens** → criar um token "Object Read & Write" **só para esse bucket**. Guarda o Access Key ID e o Secret.
3. No teu computador: `age-keygen -o ryze-backup.key`. O ficheiro contém a chave privada e mostra a pública (`age1...`). Guarda a privada em segurança e copia só a pública para o servidor.
4. No servidor, como `deploy`: `rclone config` → novo remote `r2`, tipo "S3", provider "Cloudflare", com a chave e o segredo do passo 2 e o endpoint `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`.
5. `/opt/ryze/backup.env` (600, dono `deploy`):

```
BACKUP_AGE_RECIPIENT=age1...a-tua-chave-publica...
BACKUP_R2_REMOTE=r2:ryze-backups
BACKUP_HEALTHCHECK_URL=https://hc-ping.com/<id>   # opcional (§13)
```

6. Agendar (`sudo -u deploy crontab -e`):

```
30 3 * * * /opt/ryze/deploy/backup.sh production daily >> /opt/ryze/backups/cron.log 2>&1
0 4 * * * /opt/ryze/deploy/backup.sh staging daily >> /opt/ryze/backups/cron.log 2>&1
```

### 12.4 Comandos

```bash
sudo -u deploy /opt/ryze/deploy/backup.sh production daily     # criar agora
sudo -u deploy rclone ls r2:ryze-backups/production/daily/      # listar
sudo -u deploy rclone copy r2:ryze-backups/production/daily/<ficheiro> /tmp/   # descarregar
```

**Teste de restauro** (todos os meses, sem tocar em nada real):

```bash
scp ryze-backup.key ubuntu@<IP>:/tmp/k                         # do teu computador
sudo -u deploy /opt/ryze/deploy/restore.sh test /tmp/<ficheiro-db>.sql.gz.age /tmp/k
shred -u /tmp/k                                                # apagar a chave privada
```

O teste cria um PostgreSQL temporário, restaura o backup, mostra o número de utilizadores e compras e a versão do esquema, e apaga o temporário.

### 12.5 Recuperar se o servidor desaparecer

1. Contrata um VPS novo e prepara-o como em §6 e §7. Usa **as mesmas** credenciais de base de dados que estavam no `db.env`, se as tiveres guardadas; senão, cria novas e atualiza o `app.env`.
2. Recria o `app.env` a partir do teu gestor de palavras-passe.
3. Faz o deploy da última tag boa (`deploy.sh production <tag>`). Cria uma base vazia e aplica as migrações.
4. Restaura a base:
   `restore.sh database production <backup-db> <chave> --yes`
5. Restaura os uploads:
   `restore.sh uploads production <backup-uploads> <chave> --yes`
6. Atualiza o DNS `api` (e `api-staging`) para o IP novo e o secret `VPS_HOST`/`VPS_SSH_KNOWN_HOSTS` no GitHub.
7. Verifica `/api/v1/health`, um login e uma compra de teste em staging.

---

## 13. Monitorização

| Verificar | Como | Grátis |
|---|---|---|
| Site no ar | UptimeRobot: monitor HTTP para `https://ryzecoaching.com` a cada 5 min, alerta por email | sim **[confirmar limites]** |
| API e base de dados | Monitor para `https://api.ryzecoaching.com/api/v1/health` (falha se a base não responder) | sim |
| Backups | Healthchecks.io: um "check" diário. O `backup.sh` faz ping em caso de sucesso e `/fail` em caso de erro, e o serviço avisa se o ping não chegar | sim **[confirmar limites]** |
| Erros da API | `docker compose ... logs --since 1h backend \| grep -iE "error\|panic"` | — |
| Disco | `df -h /` (alerta acima de 80%) | — |
| Memória | `free -h` e `docker stats --no-stream` | — |
| Versão | `curl https://api.ryzecoaching.com/api/v1/version` | — |

Verificação rápida semanal no servidor:

```bash
df -h / ; free -h ; docker ps --format 'table {{.Names}}\t{{.Status}}' ; tail -n 5 /opt/ryze/backups/cron.log
```

---

## 14. Credenciais e variáveis de ambiente

Os nomes foram tirados do código (`config/env.go`, `main.go`, `vite.config.ts`, `http_client.ts` e scripts). **Nunca** escrevas valores neste ficheiro.

Ambiente: **D** desenvolvimento · **S** staging · **P** produção.

| Variável | Serviço | Obrigatória? | Onde obter | Ambiente |
|---|---|---|---|---|
| `DB_NAME`, `DB_USER` | PostgreSQL | **Obrigatória** | tu escolhes (igual ao `db.env`) | D S P |
| `DB_PASSWORD` | PostgreSQL | Obrigatória fora do local | `openssl rand -base64 36` | D S P |
| `DB_HOST`, `DB_PORT`, `DB_SSLMODE` | PostgreSQL | Opcional (omissão: 127.0.0.1, 5432, disable); definidas pelo Compose | — | D (S P automático) |
| `PORT` | API | Opcional (8080) | — | D |
| `JWT_SECRET` | Sessões | **Obrigatória** (comprimento mínimo) | `openssl rand -hex 48`, um por ambiente | D S P |
| `JWT_ACCESS_TOKEN_TTL` | Sessões | Opcional (15 min) | — | D S P |
| `COOKIE_SECURE` | Sessões | Opcional (true); `false` só em local HTTP | — | D |
| `CORS_ALLOWED_ORIGINS` | API | **Obrigatória em S e P** (a omissão é localhost) | o endereço do site | S P |
| `ADMIN_1_USERNAME`, `ADMIN_1_PASSWORD`, `ADMIN_1_ACCESS_CODE` | Admin técnico | **Obrigatória** | tu escolhes (aleatórias) | D S P |
| `ADMIN_2_USERNAME`, `ADMIN_2_PASSWORD`, `ADMIN_2_ACCESS_CODE` | Admin de gestão | **Obrigatória** | idem | D S P |
| `MIN_PROGRAM_PRICE_MINOR_UNITS` | Planos | Opcional (100) | — | D S P |
| `DEFAULT_PLATFORM_COMMISSION_BPS` | Comissões | Opcional (3000 = 30%) | decisão comercial | S P |
| `STRIPE_SECRET_KEY` | Stripe | Só para cartão/MB WAY | Stripe → Developers → API keys (test em S, live em P) | S P |
| `STRIPE_PUBLISHABLE_KEY` | Stripe | Só para o checkout embutido | idem | S P |
| `STRIPE_WEBHOOK_SECRET` | Stripe | **Necessária para cartão/MB WAY** (vazia desliga-os) | Stripe → Webhooks → endpoint `https://api.<domínio>/api/v1/webhooks/stripe` | S P |
| `STRIPE_SUCCESS_URL`, `STRIPE_CANCEL_URL` | Stripe | Só para pagamentos | modelos em `docs/how_to_run.MD` com o domínio real | S P |
| `PAYPAL_CLIENT_ID`, `PAYPAL_SECRET` | PayPal | Só para PayPal | PayPal Developer → Apps | S P |
| `PAYPAL_MODE` | PayPal | Opcional (sandbox) | `live` em P | S P |
| `PAYPAL_RETURN_URL`, `PAYPAL_CANCEL_URL` | PayPal | Só para PayPal | modelos com o domínio real | S P |
| `PAYPAL_WEBHOOK_ID` | PayPal | Só para PayPal | PayPal → Webhooks → `.../api/v1/webhooks/paypal` | S P |
| `MAIL_SMTP_HOST`, `MAIL_SMTP_PORT`, `MAIL_SMTP_USERNAME`, `MAIL_SMTP_PASSWORD`, `MAIL_FROM` | Resend (SMTP) | Só para emails (definir em conjunto com `PUBLIC_APP_URL`) | Resend: host `smtp.resend.com`, porta 587, utilizador `resend`, palavra-passe = chave API **[confirmar na documentação do Resend]**; `MAIL_FROM` num domínio verificado | S P |
| `PUBLIC_APP_URL` | Emails, Google | Necessária com email ou Google | o endereço do site | S P |
| `CONTACT_SMTP_HOST`, `CONTACT_SMTP_PORT`, `CONTACT_SMTP_USERNAME`, `CONTACT_SMTP_PASSWORD`, `CONTACT_FROM`, `CONTACT_RECIPIENT` | Formulário de contacto | Só para o formulário (em conjunto) | pode usar o mesmo SMTP do Resend | S P |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URL` | Google | Só para login com Google (em conjunto) | Google Cloud → Credentials → OAuth client (web). Redirect: `https://api.<domínio>/api/v1/auth/google/callback` | S P |
| `INVOICE_SELLER_NAME`, `INVOICE_SELLER_TAX_ID`, `INVOICE_SELLER_ADDRESS` | Comprovativos | Opcional (nome "RYZE") | dados da empresa, quando existir | P |
| `TEST_MODE_ENABLED` | Modo de Teste | Opcional (false) | **false em P** | D S |
| `EXERCISE_MEDIA_ENABLED`, `EXERCISE_MEDIA_BASE_PATH` | Media de exercícios | Opcional (false) | **false até haver licença** | D S P |
| `UPLOADS_DIR`, `UPLOADS_MAX_IMAGE_BYTES` | Uploads | Opcional; diretório definido pelo Compose | — | D (S P automático) |
| `TRAINER_EVIDENCE_DIR`, `TRAINER_EVIDENCE_MAX_BYTES` | Documentos dos treinadores | Opcional; definido pelo Compose | — | D (S P automático) |
| `SERVER_READ_HEADER_TIMEOUT`, `SERVER_READ_TIMEOUT`, `SERVER_WRITE_TIMEOUT`, `SERVER_IDLE_TIMEOUT`, `SERVER_SHUTDOWN_TIMEOUT`, `SERVER_MAX_BODY_BYTES` | Servidor HTTP | Opcional (omissões seguras) | — | D S P |
| `RATE_LIMIT_ENABLED`, `RATE_LIMIT_MAX_KEYS` | Proteção | Opcional (ligado, 10 000) | — | D S P |
| `RATE_LIMIT_TRUSTED_PROXIES` | Proteção | Definida pelo Compose (rede do Caddy) | — | (S P automático) |
| `RYZE_DEMO_CLIENT_EMAIL`, `RYZE_DEMO_CLIENT_USERNAME`, `RYZE_DEMO_CLIENT_PASSWORD` | Dados de demonstração | Só `seed-demo` local | — | D |
| `TEST_USER_EMAIL`, `TEST_USER_PASSWORD` | `start_test_mode.sh` | Só local | — | D |
| `BACKUP_DIR`, `BACKUP_GPG_RECIPIENT`, `BACKUP_RETENTION_DAYS` | `scripts/backup_db.sh` (local) | Opcional | — | D |
| `BACKUP_AGE_RECIPIENT`, `BACKUP_R2_REMOTE`, `BACKUP_HEALTHCHECK_URL` | `deploy/backup.sh` (`/opt/ryze/backup.env`) | Obrigatórias para os backups no VPS (a última é opcional) | §12.3 | S P |
| `VITE_API_BASE_URL` | Build do frontend | **Obrigatória no build de S e P**; **não definir em local** (vazia parte o local) | Variable do Environment do GitHub | S P |
| `RYZE_LAN`, `RYZE_LAN_BACKEND` | Vite (telemóvel na rede local) | Opcional | — | D |
| `RYZE_VERSION`, `RYZE_DOCKER_SUBNET` | `docker-compose.yml` local | Opcional | — | D |

**Variáveis que podem ficar vazias sem impedir o arranque:**

- Pagamentos, email, Google, contacto, faturação e media.
- A API arranca e desliga a função correspondente:
  - os métodos de pagamento por configurar mostram um erro e não criam a compra;
  - os emails ficam registados como "skipped";
  - o botão do Google não funciona.
- Grupos que têm de ser preenchidos **juntos**, senão a API recusa arrancar com uma mensagem clara: Google (todos com `PUBLIC_APP_URL`), Mail (`MAIL_SMTP_HOST`, `MAIL_FROM`, `PUBLIC_APP_URL`), Contacto.

**Onde fica cada credencial e quando a inseres:**

| Onde | O quê | Quando |
|---|---|---|
| `.env` no teu computador | só desenvolvimento (chaves de teste) | já |
| `/opt/ryze/staging/app.env`, `db.env` | segredos de staging (Stripe test, PayPal sandbox) | antes do primeiro deploy de staging |
| `/opt/ryze/production/app.env`, `db.env` | segredos de produção (live) | antes de abrir ao público e de aceitar pagamentos |
| `/opt/ryze/backup.env` | chave pública `age`, remote R2 | antes do primeiro deploy |
| GitHub → Environments | SSH, Cloudflare token e IDs, `VITE_API_BASE_URL` | antes do primeiro deploy automático |
| Gestor de palavras-passe (o teu) | **tudo** o que está acima + chave privada `age` + chaves SSH | sempre |

**Nunca** uses o mesmo `JWT_SECRET`, as mesmas palavras-passe de admin ou de base de dados em ambientes diferentes.

---

## 15. Testes antes do lançamento

**Infraestrutura**

- [ ] `https://ryzecoaching.com` abre; `http://` e `www.` redirecionam.
- [ ] `https://api.ryzecoaching.com/api/v1/health` devolve `ok`; `/version` mostra a versão certa.
- [ ] A porta 5432 não responde de fora: `nc -vz <IP> 5432` falha.
- [ ] SSH com palavra-passe é recusado.
- [ ] O certificado é válido (cadeado) e o modo da Cloudflare é Full (strict).

**Aplicação** (primeiro em staging)

- [ ] Registo, email de verificação recebido, login, logout, recuperação de palavra-passe.
- [ ] Login com Google.
- [ ] Upload de avatar e de capa; as imagens aparecem e continuam lá depois de um novo deploy.
- [ ] Plano grátis: ativar, treinar, histórico, progresso.
- [ ] Compra com cartão de teste do Stripe (incluindo 3-D Secure), MB WAY de teste e PayPal sandbox; o webhook conclui a compra; chega o comprovativo; o acesso abre.
- [ ] Reembolso no admin retira o acesso.
- [ ] Código de desconto aplicado no pagamento.
- [ ] Premium Nível 2: questionário, oferta ao treinador, entrega.
- [ ] Admins: login com código de acesso nos dois consoles.
- [ ] PWA: instalar no telemóvel; o aviso de nova versão aparece depois de um deploy.
- [ ] Aviso de cookies, páginas legais e rodapé.
- [ ] Telemóvel (360–430 px), tablet e computador.

**Operação**

- [ ] Backup diário no R2 e **teste de restauro** feito com sucesso.
- [ ] Rollback testado: publicar uma tag antiga em staging e voltar.
- [ ] Monitores e alertas a funcionar (desliga a API de staging e confirma que chega o alerta).

**Carga** (staging, antes de abrir ao público). Exemplo com [k6](https://k6.io), corrido do teu computador:

```js
// k6 run carga.js
import http from "k6/http";
export const options = { vus: 50, duration: "2m" };
export default () => { http.get("https://api-staging.ryzecoaching.com/api/v1/programs?page=1&limit=12"); };
```

Durante o teste, vê no servidor `docker stats`: CPU, RAM e erros. Repete com mais utilizadores virtuais até os tempos subirem. O resultado é a base real para decidir quando aumentar o VPS (§19).

> **Atenção:** o rate limiting da API vai limitar um único IP. Em testes de carga a partir de uma só máquina, espera respostas `429`; são esperadas. Para medir a capacidade, usa endpoints públicos de leitura e lê esses `429` como "proteção a funcionar".

---

## 16. Custos

Preços de 2026-11-27, com IVA de Portugal (23%) quando a fonte o dá. **[confirmar]** = a validar no carrinho.

| Serviço | Inicial | Mensal | Anual | Notas |
|---|---|---|---|---|
| Domínio `.com` (Cloudflare) | ~11,17 USD | — | ~11,17 USD (~10–11 €) **[confirmar IVA e câmbio]** | ao custo; renovação ao mesmo preço |
| VPS OVH VPS-2 | 0 € | 8,87 € | 106,44 € | com IVA; tráfego ilimitado; backup diário OVH incluído |
| Cloudflare DNS, proxy, TLS (Free) | 0 | 0 | 0 | |
| Cloudflare Pages (Free) | 0 | 0 | 0 | 500 builds/mês |
| Cloudflare R2 | 0 | 0 € até 10 GB | 0 | depois 0,015 USD/GB-mês; saída de dados grátis |
| GitHub (repositório público) | 0 | 0 | 0 | Actions grátis em repositórios públicos; GHCR **[confirmar limites para imagens públicas]** |
| Resend (Free) | 0 | 0 | 0 | 3 000 emails/mês, **100/dia**; Pro 20 USD/mês (50 000) quando precisares |
| Stripe / PayPal | 0 | comissão por transação | — | **[confirmar tarifas em vigor para PT/UE]** |
| Monitorização (UptimeRobot, Healthchecks.io) | 0 | 0 | 0 | planos gratuitos **[confirmar]** |

**Três cenários (mensal, aproximado):**

| Cenário | Composição | Mensal |
|---|---|---|
| 1. Mais económico | OVH VPS-1 (4,69 €) + domínio (~0,9 €/mês) + tudo grátis | **≈ 5,6 €** |
| 2. Recomendado | OVH VPS-2 (8,87 €) + domínio + tudo grátis | **≈ 9,8 €** |
| 3. Preparado para crescimento | OVH VPS-3 (12,79 €) + domínio + Resend Pro (≈ 18,5 €) | **≈ 32 €** (o Resend Pro só quando passarem 100 emails/dia) |

Os cenários 1 e 2 ficam abaixo do orçamento de 15–25 €. O cenário 3 só se justifica com crescimento real.

Custos variáveis a vigiar:

- emails acima do plano gratuito;
- R2 acima de 10 GB (só se os uploads passarem a viver lá);
- comissões de pagamento.

---

## 17. Situações de emergência

| Situação | O que fazer |
|---|---|
| **O servidor caiu** (não responde ao SSH) | Painel da OVH → estado do VPS → reiniciar. Se não voltar, abre um pedido de suporte à OVH. Se o servidor se perdeu, §12.5. |
| **A API parou** (`/health` falha) | `docker ps` → o container `ryze-production-backend-1` está a reiniciar? Vê `logs --tail 200 backend`. Erro de configuração: corrige o `app.env` e `docker compose ... up -d backend`. Erro depois de um deploy: `deploy.sh production <tag-anterior>`. |
| **O PostgreSQL não arranca** | `logs db`. Disco cheio (`df -h`) → limpa `docker image prune`, os backups locais antigos e os logs. Base corrompida → restauro (§12.5, passo 4) a partir do último backup. |
| **O domínio deixou de funcionar** | Cloudflare → o domínio está ativo e renovado? (A renovação falha se o cartão expirou.) DNS: os registos `api` e do Pages existem? |
| **Um deploy falhou** | O GitHub mostra o passo. Testes → corrige e volta a fazer push. Deploy no VPS → o script voltou sozinho à imagem anterior; lê `deployments.log`. |
| **Uma migração falhou** | A API ficou na versão anterior. A base pode ter ficado marcada como "dirty". **Não** tentes `migrate down`. Lê o erro, restaura o backup `pre-deploy` dessa hora (`restore.sh database ...`), corrige a migração no código e faz novo deploy. |
| **Não existe backup** | Corre já `backup.sh production daily` e confirma no R2. Vê o `cron.log` e o Healthchecks para perceber desde quando falha. |
| **Uma credencial expirou ou foi exposta** | Gera uma nova no fornecedor, atualiza o `app.env` (ou o secret do GitHub), `docker compose ... up -d backend` e revoga a antiga. Se foi o `JWT_SECRET`, todas as sessões terminam (é o esperado). |

---

## 18. Preparação comercial e legal

> Esta secção **não é aconselhamento jurídico nem fiscal**. Identifica o que tem de ser confirmado com um contabilista certificado e um advogado **antes** de receber pagamentos ou prestar serviços comercialmente.

**Publicar o site tecnicamente** (staging, ou produção sem vendas):

- não depende da empresa;
- **pressupõe** uma pessoa com capacidade legal para contratar o domínio e o VPS.

**Operar comercialmente** (vender planos, pagar treinadores) depende de:

1. **Titular responsável.**
   - Stripe e PayPal exigem a verificação de identidade de um titular adulto ou de uma entidade, e uma conta bancária no nome dele.
   - **Se ainda não és maior de idade, as contas de pagamento (e idealmente a do domínio e a do VPS) têm de estar em nome de um adulto responsável ou da futura empresa.** Não se contorna.
   - Mudar mais tarde o titular de uma conta Stripe ou PayPal pode obrigar a criar uma conta nova; decide o titular antes de as abrir.
2. **Forma de atividade.**
   - Vender sem empresa constituída obriga a outra forma legal de faturar, por exemplo atividade aberta nas Finanças em nome do titular.
   - **Confirmar com um contabilista** qual a forma adequada e desde quando.
3. **Faturação certificada.**
   - A app emite um comprovativo **sem valor fiscal**.
   - Para vendas reais é preciso fatura emitida por software certificado pela AT e comunicada à AT. O código está preparado para ligar um fornecedor (InvoiceXpress, Moloni, Vendus...), mas a integração não está feita.
4. **IVA.**
   - O regime de isenção (art. 53.º) tem limites.
   - Vendas de serviços digitais a consumidores de outros países da UE podem obrigar ao regime OSS.
   - **Confirmar com o contabilista.**
5. **Identidade nos documentos legais.**
   - `main/src/frontend/src/constants/legal.ts` tem campos por preencher: nome, NIF/NIPC, morada, registo e capital.
   - Sem eles, os Termos, a Privacidade e o Aviso Legal não estão completos.
6. **Livro de Reclamações Eletrónico.** Registo obrigatório da entidade que vende.
7. **RGPD.**
   - Rever com um advogado a Política de Privacidade.
   - Confirmar os acordos de tratamento de dados com os subcontratantes: OVH, Cloudflare, Resend, Google, Stripe, PayPal.
   - Os dados ficam na UE (VPS e R2 na UE), exceto onde os fornecedores indicarem outra coisa.
8. **Media dos exercícios.**
   - As imagens e animações do conjunto "© Gym visual" **não podem ser servidas publicamente** sem licença comercial confirmada.
   - Enquanto isso: `EXERCISE_MEDIA_ENABLED=false`. O treino mostra o estado "media indisponível".
9. **Treinadores.**
   - Contratos de prestação de serviço e de confidencialidade (veem dados de saúde).
   - Pagamento das comissões (hoje só são calculadas).
10. **Idade dos utilizadores.** Os Termos definem 16 anos; confirmar.

---

## 19. Escalabilidade

- **Contas registadas não são utilizadores simultâneos.** 2 000 contas costumam ser algumas dezenas de pessoas ativas ao mesmo tempo. Decide com medições (§15), não com estimativas.
- **Sinais para crescer:**
  - CPU média acima de 70% ou RAM acima de 80% de forma sustentada;
  - tempos de resposta da API acima de 500 ms nas listagens;
  - disco acima de 70%;
  - o teste de carga a falhar antes de 2–3 vezes o pico real observado.
- **Passo 1, vertical:** VPS-2 → VPS-3 (mesmos dados, minutos de paragem).
- **Passo 2, separar o PostgreSQL** (segundo VPS na rede privada, ou PostgreSQL gerido):
  - exige mudar só `DB_HOST` e `DB_SSLMODE`;
  - cada instância da API abre até **25 ligações** (constante em `main/src/backend/database/database.go`). Com várias instâncias, a soma tem de ficar abaixo do `max_connections` do PostgreSQL (100 por omissão), por isso a partir de 4 instâncias é preciso baixar esse valor ou usar um pooler (PgBouncer).
- **Passo 3, várias instâncias da API** atrás do Caddy (load balancing). Antes disso são precisas três alterações de código:
  1. **uploads em armazenamento partilhado** (R2/S3). Hoje vivem no disco local; ver nota abaixo;
  2. **rate limiting partilhado** (hoje em memória, por instância). Sem isto, cada instância conta à parte;
  3. as **sessões** já funcionam: são JWT validados com a base e não precisam de memória partilhada. Os jobs em background já suportam várias instâncias.
- **R2 para uploads:**
  - só faz sentido com várias instâncias, ou se o disco ficar curto;
  - exige um armazenamento "S3-compatível" no lugar de `services/public_uploads` (ficheiros locais);
  - os ficheiros privados (fotos de treino, documentos dos treinadores) ficariam num bucket **privado** com URLs temporários;
  - **não está implementado**. Para começar, o disco do VPS com backup para o R2 é mais simples e suficiente.
- **Sem Kubernetes**: não há necessidade demonstrada.

---

## 20. Onde estamos e caminho até ao lançamento

### 20.1 Respostas diretas

1. **Fase:** fim do desenvolvimento das funções principais. A infraestrutura está documentada e com modelos prontos, mas nada contratado.
2. **Completamente preparado:**
   - código da aplicação e testes (backend em base nova, frontend e build);
   - migrações;
   - modelos de deploy;
   - CI com base de dados;
   - workflow de deploy;
   - scripts de backup e restauro.
3. **Precisa de implementação antes de vender:**
   - integração de faturação certificada;
   - pagamento das comissões aos treinadores (processo);
   - R2 para uploads (só ao crescer).
4. **Só configuração:** domínio, DNS, HTTPS, servidor, segredos por ambiente, Pages, R2 para backups, monitorização.
5. **Credenciais a obter:** Cloudflare (token do Pages, Account ID, R2), Resend, Google OAuth, Stripe (test e live), PayPal (sandbox e live).
6. **Serviços a contratar:** o domínio (Cloudflare) e o VPS (OVH VPS-2). Tudo o resto começa grátis.
7. **O que podes fazer sem empresa:**
   - ler e rever este manual;
   - criar a conta Cloudflare;
   - preparar as chaves SSH e a chave `age`;
   - criar contas Resend e Google e testá-las em staging;
   - Stripe e PayPal **em modo de teste/sandbox**;
   - montar staging.
8. **Precisa de um adulto responsável ou de confirmação externa:**
   - contratos de domínio e VPS (capacidade legal);
   - ativação live do Stripe e do PayPal (verificação de identidade);
   - forma de faturação e IVA (contabilista);
   - documentos legais (advogado);
   - licença dos media.
9. **Testes que faltam:**
   - primeira execução real do CI e do deploy no GitHub;
   - build da imagem;
   - testes de pagamento ponta a ponta com webhooks reais (test mode);
   - teste de restauro;
   - teste de carga;
   - rollback;
   - verificação em telemóveis reais com HTTPS.
10. **Caminho mais curto e seguro até ao primeiro deploy:**
    1. comprar o domínio;
    2. contratar o VPS-2;
    3. preparar o servidor (§6);
    4. criar o R2 e a chave `age` (§12.3);
    5. criar os Environments no GitHub (§11.3);
    6. criar o ramo `staging` e fazer push;
    7. testar o staging (§15);
    8. só depois produção.

### 20.2 Checklists

**Antes de comprar o domínio**
- [ ] Decidir o titular das contas (§18.1).
- [ ] Confirmar o nome final da marca.
- [ ] Rever o preço no carrinho da Cloudflare.

**Antes de contratar o VPS**
- [ ] Confirmar plano e preço no carrinho da OVH (VPS-2, UE, Ubuntu 24.04).
- [ ] Criar as chaves SSH (§6.1).

**Antes do primeiro deploy**
- [ ] Servidor preparado (§6), PostgreSQL configurado (§7), `app.env` e `db.env` de staging criados.
- [ ] Certificado de origem instalado (§10.2) e registos DNS criados (§10.1).
- [ ] R2, rclone, chave `age` e cron de backups (§12.3).
- [ ] Environments, secrets e variables no GitHub (§11.3); projeto Pages criado (§9).

**Antes de abrir o staging**
- [ ] Cloudflare Access a proteger o staging.
- [ ] Stripe em modo de teste, PayPal sandbox, Resend com domínio verificado.
- [ ] Primeiro teste de restauro.

**Antes de abrir o site ao público**
- [ ] Toda a checklist de §15 em staging.
- [ ] Teste de carga e monitores a funcionar.
- [ ] Documentos legais completos e revistos (§18.5, §18.7).
- [ ] `TEST_MODE_ENABLED=false`, `EXERCISE_MEDIA_ENABLED=false` (ou licença confirmada), segredos de produção novos.

**Antes de aceitar pagamentos**
- [ ] Titular verificado no Stripe e no PayPal (live).
- [ ] Forma de faturação decidida e integração de faturação certificada.
- [ ] IVA confirmado com contabilista.
- [ ] Livro de Reclamações registado.
- [ ] Webhooks live configurados e uma compra real pequena + reembolso testados.

**Depois do lançamento**
- [ ] Backups verificados todas as semanas; teste de restauro todos os meses.
- [ ] Atualizações do servidor (automáticas) e reinício mensal planeado.
- [ ] Rever custos e métricas todos os meses (§19).
- [ ] Renovação do domínio e do certificado de origem (15 anos) anotadas.

---

### Ficheiros de infraestrutura no repositório

| Ficheiro | Para quê |
|---|---|
| `deploy/compose.app.yml` | Stack de um ambiente no VPS (API + PostgreSQL, sem portos públicos) |
| `deploy/compose.edge.yml` | Caddy, o único com portos 80/443 |
| `deploy/Caddyfile` | HTTPS com o certificado de origem; IP real dos visitantes vindo da Cloudflare |
| `deploy/deploy.sh` | Deploy com backup, migrações, health check e rollback |
| `deploy/backup.sh` / `deploy/restore.sh` | Backups cifrados para o R2; teste de restauro e restauro |
| `deploy/ryze-backend.service` | Alternativa sem Docker (systemd) |
| `.github/workflows/ci.yml` | Testes (agora com PostgreSQL) |
| `.github/workflows/deploy.yml` | Deploy de staging e produção |
| `public/_headers` | Cabeçalhos de segurança e cache no Cloudflare Pages |
| `docker-compose.yml` | Stack local completa (desenvolvimento ou teste) |

**Fontes dos preços (consultadas em 2026-11-27):**

- OVHcloud VPS: https://www.ovhcloud.com/pt/vps/ e https://blog.ovhcloud.com/en/posts/pricing-evolution-of-public-cloud-bare-metal-and-vps-at-ovhcloud/
- Cloudflare Registrar: https://developers.cloudflare.com/registrar/ e https://actualdatabase.com/domain-price-per-year/
- Cloudflare Pages (limites): https://developers.cloudflare.com/pages/platform/limits
- Cloudflare R2 (preços): https://developers.cloudflare.com/r2/pricing
- Resend (preços): https://resend.com/pricing.md e https://resend.com/docs/knowledge-base/resend-sending-limits
- Hetzner (preços): https://cloudtally.eu/hetzner-cloud-prices
- GitHub Actions: https://docs.github.com/en/billing/concepts/product-billing/github-actions
- RDAP `.com`: https://rdap.verisign.com/com/v1/domain/ryzecoaching.com
