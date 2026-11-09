from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.on_demand_engine_order_input import OnDemandEngineOrderInput
from ...models.voice_engine_order_results import VoiceEngineOrderResults
from typing import cast



def _get_kwargs(
    *,
    body: OnDemandEngineOrderInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/voice-engines/on-demand-orders",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceEngineOrderResults:
    if response.status_code == 200:
        response_200 = VoiceEngineOrderResults.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceEngineOrderResults]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: OnDemandEngineOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceEngineOrderResults]:
    """  Administrateur : ordonne l'installation d'un engine à la demande (famille de bibliothèque et
    version, version par défaut de la famille si omise) sur un ou plusieurs Workers. Aucune enveloppe
    signée n'est acceptée : l'identité est dérivée de la famille, de la version et de la tâche. Le
    matériel du Worker est contrôlé avant tout ordre ; clé idempotente, génération, livraison, accusé,
    inventaire et causes d'échec sont ceux des ordres d'engine signés. Famille inconnue : 400
    unsupported_engine_family.

    Args:
        origin (str):
        x_csrf_token (str):
        body (OnDemandEngineOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceEngineOrderResults]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient | Client,
    body: OnDemandEngineOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceEngineOrderResults | None:
    """  Administrateur : ordonne l'installation d'un engine à la demande (famille de bibliothèque et
    version, version par défaut de la famille si omise) sur un ou plusieurs Workers. Aucune enveloppe
    signée n'est acceptée : l'identité est dérivée de la famille, de la version et de la tâche. Le
    matériel du Worker est contrôlé avant tout ordre ; clé idempotente, génération, livraison, accusé,
    inventaire et causes d'échec sont ceux des ordres d'engine signés. Famille inconnue : 400
    unsupported_engine_family.

    Args:
        origin (str):
        x_csrf_token (str):
        body (OnDemandEngineOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceEngineOrderResults
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient | Client,
    body: OnDemandEngineOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceEngineOrderResults]:
    """  Administrateur : ordonne l'installation d'un engine à la demande (famille de bibliothèque et
    version, version par défaut de la famille si omise) sur un ou plusieurs Workers. Aucune enveloppe
    signée n'est acceptée : l'identité est dérivée de la famille, de la version et de la tâche. Le
    matériel du Worker est contrôlé avant tout ordre ; clé idempotente, génération, livraison, accusé,
    inventaire et causes d'échec sont ceux des ordres d'engine signés. Famille inconnue : 400
    unsupported_engine_family.

    Args:
        origin (str):
        x_csrf_token (str):
        body (OnDemandEngineOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceEngineOrderResults]
     """


    kwargs = _get_kwargs(
        body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient | Client,
    body: OnDemandEngineOrderInput,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceEngineOrderResults | None:
    """  Administrateur : ordonne l'installation d'un engine à la demande (famille de bibliothèque et
    version, version par défaut de la famille si omise) sur un ou plusieurs Workers. Aucune enveloppe
    signée n'est acceptée : l'identité est dérivée de la famille, de la version et de la tâche. Le
    matériel du Worker est contrôlé avant tout ordre ; clé idempotente, génération, livraison, accusé,
    inventaire et causes d'échec sont ceux des ordres d'engine signés. Famille inconnue : 400
    unsupported_engine_family.

    Args:
        origin (str):
        x_csrf_token (str):
        body (OnDemandEngineOrderInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceEngineOrderResults
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
