"""
Abstração do gateway de pagamento.

Conforme PRD seção 9.2: o código de negócio (routers, repositórios) nunca
chama o SDK do Stripe diretamente. Toda interação passa por este módulo.

A implementação concreta é selecionada pela variável BILLING_PROVIDER.
Para trocar de gateway, basta criar billing_pagarme.py e mudar o env var.
"""

import os

BILLING_PROVIDER = os.getenv("BILLING_PROVIDER", "stripe")

if BILLING_PROVIDER == "stripe":
    from app.services.billing_stripe import (  # noqa: F401
        create_checkout_session,
        get_or_create_customer,
        verify_webhook_signature,
    )
else:
    raise NotImplementedError(f"BILLING_PROVIDER '{BILLING_PROVIDER}' não suportado.")
