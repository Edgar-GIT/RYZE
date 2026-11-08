import { ArrowLeft, CheckCircle2, ClipboardList, FlaskConical, ShieldCheck, UserRound } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useHistory, useLocation } from "react-router-dom";

import { Button } from "@/components/button/button";
import { Container } from "@/components/container/container";
import { PageWrapper } from "@/components/page_wrapper/page_wrapper";
import { useTestMode } from "@/components/test_mode/test_mode_context";
import { useProgramCheckout } from "@/hooks/use_program_checkout";
import { formatMarketplacePrice, type MarketplaceProgram } from "@/services/marketplace_api";
import {
  fetchMyPremiumLevel2Requests,
  fetchPremiumLevel2Programs,
  PREMIUM_LEVEL_2_STATUS_LABELS,
  type PremiumLevel2Request
} from "@/services/premium_level_2_api";
import { ApiError } from "@utils/http_client";

import styles from "./premium_level2_page.module.css";

const INCLUDES = [
  "A detailed intake reviewed by a certified RYZE trainer",
  "An individual training programme authored for your goal and schedule",
  "An individual nutrition plan with meals, portions and hydration guidance",
  "Direct delivery into My Programs when your trainer completes the work"
];

const editableStatus = (status: PremiumLevel2Request["status"]): boolean =>
  status === "awaiting_intake" || status === "awaiting_trainer_assignment";

export const PremiumLevel2Page = () => {
  const history = useHistory();
  const location = useLocation();
  const { isActive: testModeActive } = useTestMode();

  const [offers, setOffers] = useState<MarketplaceProgram[]>([]);
  const [requests, setRequests] = useState<PremiumLevel2Request[]>([]);
  const [loading, setLoading] = useState(true);
  const [errorMessage, setErrorMessage] = useState("");

  const requestedProgramId = new URLSearchParams(location.search).get("program") ?? "";
  const returningFromPayment = new URLSearchParams(location.search).has("purchase_id");

  const load = useCallback(async () => {
    setLoading(true);
    setErrorMessage("");
    try {
      const [published, own] = await Promise.all([
        fetchPremiumLevel2Programs(),
        fetchMyPremiumLevel2Requests().catch((error: unknown) => {
          if (error instanceof ApiError && error.status === 401) {
            return [];
          }
          throw error;
        })
      ]);
      setOffers(published);
      setRequests(own);
      if (published.length === 0) {
        setErrorMessage("Premium Level 2 is not available right now. Please check back soon.");
      }
    } catch {
      setOffers([]);
      setRequests([]);
      setErrorMessage("We could not load Premium Level 2. Please try again.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const selectedOffer = useMemo(() => {
    if (offers.length === 0) return null;
    if (requestedProgramId) {
      return offers.find((offer) => offer.id === requestedProgramId) ?? null;
    }
    return null;
  }, [offers, requestedProgramId]);

  const selectedRequest = useMemo(
    () =>
      selectedOffer
        ? requests.find((request) => request.program_id === selectedOffer.id) ?? null
        : null,
    [requests, selectedOffer]
  );

  const checkout = useProgramCheckout({
    programId: selectedOffer?.id ?? "",
    free: false,
    testModeActive,
    enabled: selectedOffer !== null && !loading
  });

  const refreshRequests = useCallback(async (): Promise<PremiumLevel2Request[]> => {
    try {
      const own = await fetchMyPremiumLevel2Requests();
      setRequests(own);
      return own;
    } catch {
      return [];
    }
  }, []);

  const goToRequest = useCallback(
    (request: PremiumLevel2Request) => {
      history.replace(`/services/premium-level-2/requests/${request.id}`);
    },
    [history]
  );

  useEffect(() => {
    if (checkout.state.status !== "success" || !selectedOffer) {
      return;
    }
    let cancelled = false;
    void refreshRequests().then((own) => {
      if (cancelled) return;
      const created = own.find((request) => request.program_id === selectedOffer.id);
      if (created) {
        goToRequest(created);
      }
    });
    return () => {
      cancelled = true;
    };
  }, [checkout.state, goToRequest, refreshRequests, selectedOffer]);

  useEffect(() => {
    if (!returningFromPayment || !selectedOffer || selectedRequest) {
      return;
    }
    void refreshRequests();
  }, [refreshRequests, returningFromPayment, selectedOffer, selectedRequest]);

  if (loading) {
    return (
      <PageWrapper className={styles.page}>
        <section className={styles.detail}>
          <Container size="wide">
            <p className={styles.stateText}>Loading Premium Level 2…</p>
          </Container>
        </section>
      </PageWrapper>
    );
  }

  if (!selectedOffer) {
    return (
      <PageWrapper className={styles.page}>
        <section className={styles.detail}>
          <Container size="wide" className={styles.container}>
            <div className={styles.back}>
              <Button to="/services" variant="ghost" size="small" icon={<ArrowLeft size={15} />} iconPosition="left">
                Back to Services
              </Button>
            </div>
            <header className={styles.heading}>
              <p className={styles.eyebrow}>
                <UserRound size={14} aria-hidden="true" />
                Premium Level 2
              </p>
              <h1 className={styles.title}>One-to-one training, built around you</h1>
              <p className={styles.subtitle}>
                Buy the service first, complete your intake, and a certified RYZE trainer will
                author your individual training and nutrition plans.
              </p>
            </header>

            {requests.length > 0 ? (
              <section className={styles.panel}>
                <h2 className={styles.sectionTitle}>Your Premium Level 2 services</h2>
                <div className={styles.requestList}>
                  {requests.map((request) => (
                    <article key={request.id} className={styles.requestRow}>
                      <div>
                        <p className={styles.requestName}>{request.offer_name}</p>
                        <p className={styles.sectionText}>
                          {PREMIUM_LEVEL_2_STATUS_LABELS[request.status]}
                        </p>
                      </div>
                      <Button
                        variant={editableStatus(request.status) ? "primary" : "secondary"}
                        size="small"
                        onClick={() => goToRequest(request)}
                      >
                        {editableStatus(request.status) ? "Continue intake" : "View service"}
                      </Button>
                    </article>
                  ))}
                </div>
              </section>
            ) : null}

            {errorMessage ? <p className={styles.errorText}>{errorMessage}</p> : null}

            <div className={styles.offerGrid}>
              {offers.map((offer) => (
                <article key={offer.id} className={styles.offerCard}>
                  <div className={styles.offerBody}>
                    <h2 className={styles.offerName}>{offer.name}</h2>
                    <p className={styles.sectionText}>{offer.description}</p>
                    <div className={styles.chips}>
                      <span className={styles.chipPrice}>
                        {testModeActive
                          ? formatMarketplacePrice(0, offer.currency, offer.type)
                          : formatMarketplacePrice(offer.price_minor_units, offer.currency, offer.type)}
                      </span>
                      {testModeActive ? (
                        <span className={styles.chipTestMode}>
                          <FlaskConical size={12} aria-hidden="true" />
                          Test Mode
                        </span>
                      ) : null}
                      <span className={styles.chip}>Trainer delivered</span>
                    </div>
                  </div>
                  <Button
                    variant="primary"
                    size="small"
                    to={`/services/premium-level-2?program=${offer.id}`}
                  >
                    View service
                  </Button>
                </article>
              ))}
            </div>
          </Container>
        </section>
      </PageWrapper>
    );
  }

  return (
    <PageWrapper className={styles.page}>
      <section className={styles.detail}>
        <Container size="wide" className={styles.container}>
          <div className={styles.back}>
            <Button
              to="/services/premium-level-2"
              variant="ghost"
              size="small"
              icon={<ArrowLeft size={15} />}
              iconPosition="left"
            >
              All Premium Level 2 services
            </Button>
          </div>

          <header className={styles.heading}>
            <p className={styles.eyebrow}>
              <ShieldCheck size={14} aria-hidden="true" />
              Premium Level 2
            </p>
            <h1 className={styles.title}>{selectedOffer.name}</h1>
            <p className={styles.subtitle}>{selectedOffer.description}</p>
            <div className={styles.chips}>
              <span className={styles.chipPrice}>
                {testModeActive
                  ? formatMarketplacePrice(0, selectedOffer.currency, selectedOffer.type)
                  : formatMarketplacePrice(
                      selectedOffer.price_minor_units,
                      selectedOffer.currency,
                      selectedOffer.type
                    )}
              </span>
              {testModeActive ? (
                <span className={styles.chipTestMode}>
                  <FlaskConical size={12} aria-hidden="true" />
                  Test Mode
                </span>
              ) : null}
              <span className={styles.chip}>One-to-one delivery</span>
            </div>
          </header>

          <div className={styles.columns}>
            <div className={styles.main}>
              <section className={styles.panel}>
                <h2 className={styles.sectionTitle}>What you receive</h2>
                <ul className={styles.includes}>
                  {INCLUDES.map((item) => (
                    <li key={item} className={styles.includeItem}>
                      <CheckCircle2 size={16} aria-hidden="true" />
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </section>

              <section className={styles.panel}>
                <h2 className={styles.sectionTitle}>How it works</h2>
                <ol className={styles.steps}>
                  <li>
                    <strong>Purchase the service.</strong> The offer itself contains no training
                    plan; your programme is created for you afterwards.
                  </li>
                  <li>
                    <strong>Complete your intake.</strong> Your trainer uses it to build around your
                    goal, experience, schedule and dietary needs.
                  </li>
                  <li>
                    <strong>Your trainer builds both plans.</strong> A RYZE administrator allocates
                    the trainer after your intake is saved.
                  </li>
                  <li>
                    <strong>Receive your plan.</strong> Training appears in My Programs and the
                    nutrition plan is available from your service page.
                  </li>
                </ol>
              </section>
            </div>

            <aside className={styles.aside}>
              <section className={styles.panel}>
                <h2 className={styles.sectionTitle}>Your service</h2>
                {selectedRequest ? (
                  <>
                    <p className={styles.sectionText}>
                      {PREMIUM_LEVEL_2_STATUS_LABELS[selectedRequest.status]}
                    </p>
                    <Button
                      variant="primary"
                      size="small"
                      icon={<ClipboardList size={15} />}
                      iconPosition="left"
                      onClick={() => goToRequest(selectedRequest)}
                    >
                      {editableStatus(selectedRequest.status) ? "Complete your intake" : "Open service"}
                    </Button>
                  </>
                ) : checkout.state.status === "success" ? (
                  <>
                    <p className={styles.successText}>
                      <CheckCircle2 size={16} aria-hidden="true" />
                      Payment confirmed
                    </p>
                    <p className={styles.sectionText}>
                      Your service is being prepared. Refresh in a moment if it does not appear.
                    </p>
                    <Button
                      variant="secondary"
                      size="small"
                      onClick={() => {
                        void refreshRequests().then((own) => {
                          const created = own.find((request) => request.program_id === selectedOffer.id);
                          if (created) goToRequest(created);
                        });
                      }}
                    >
                      Check my service
                    </Button>
                  </>
                ) : (
                  <>
                    <p className={styles.sectionText}>
                      Complete checkout to start your intake. Your trainer is allocated after the
                      intake is submitted.
                    </p>
                    {testModeActive ? (
                      <p className={styles.testModeNote}>
                        <FlaskConical size={14} aria-hidden="true" />
                        Test Mode is active. No payment provider is contacted.
                      </p>
                    ) : null}
                    {checkout.state.status === "guest" ? (
                      <Button variant="primary" size="small" to="/login">
                        Sign in to purchase
                      </Button>
                    ) : checkout.state.status === "error" ? (
                      <div className={styles.actions}>
                        <p className={styles.errorText}>{checkout.state.message}</p>
                        <Button variant="secondary" size="small" onClick={() => void checkout.buy()}>
                          Try again
                        </Button>
                      </div>
                    ) : checkout.state.status === "owned" ? (
                      <p className={styles.sectionText}>
                        This service is already active. Open it from the list above.
                      </p>
                    ) : (
                      <>
                        {checkout.paymentMethods.length > 0 ? (
                          <fieldset className={styles.methods}>
                            <legend className={styles.label}>Payment method</legend>
                            <div className={styles.methodOptions}>
                              {checkout.paymentMethods.map((method) => (
                                <label
                                  key={method.method}
                                  className={styles.method}
                                >
                                  <input
                                    type="radio"
                                    name="premium-level-2-method"
                                    value={method.method}
                                    checked={checkout.selectedMethod === method.method}
                                    onChange={() => checkout.selectMethod(method.method)}
                                  />
                                  <span>{method.label}</span>
                                </label>
                              ))}
                            </div>
                          </fieldset>
                        ) : (
                          <p className={styles.sectionText}>
                            Checkout is temporarily unavailable. No payment can be started right now.
                          </p>
                        )}
                        <Button
                          variant="primary"
                          size="small"
                          disabled={
                            checkout.paymentMethods.length === 0 ||
                            ["creating", "negotiating", "redirecting", "processing"].includes(
                              checkout.state.status
                            )
                          }
                          onClick={() => void checkout.buy()}
                        >
                          {checkout.state.status === "checking"
                            ? "Checking access…"
                            : "Buy Premium Level 2"}
                        </Button>
                      </>
                    )}
                  </>
                )}
              </section>
            </aside>
          </div>
        </Container>
      </section>
    </PageWrapper>
  );
};
