from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.fingerprint_transition_preview import FingerprintTransitionPreview
from typing import cast



def _get_kwargs(
    *,
    to: str,

) -> dict[str, Any]:




    params: dict[str, Any] = {}

    params["to"] = to


    params = {k: v for k, v in params.items() if v is not UNSET and v is not None}


    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-models/fingerprint-transition/preview",
        "params": params,
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | FingerprintTransitionPreview:
    if response.status_code == 200:
        response_200 = FingerprintTransitionPreview.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | FingerprintTransitionPreview]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    to: str,

) -> Response[Error | FingerprintTransitionPreview]:
    """  Administrateurs uniquement. Aperçu d'impact du changement du modèle d'empreinte vers le modèle
    cible, sans rien modifier : références à recalculer, clips et Personnes concernés, autres empreintes
    à recalculer ensuite, attributions validées conservées, Workers qui ont déjà la cible prête (ADR
    0044).

    Args:
        to (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FingerprintTransitionPreview]
     """


    kwargs = _get_kwargs(
        to=to,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,
    to: str,

) -> Error | FingerprintTransitionPreview | None:
    """  Administrateurs uniquement. Aperçu d'impact du changement du modèle d'empreinte vers le modèle
    cible, sans rien modifier : références à recalculer, clips et Personnes concernés, autres empreintes
    à recalculer ensuite, attributions validées conservées, Workers qui ont déjà la cible prête (ADR
    0044).

    Args:
        to (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FingerprintTransitionPreview
     """


    return sync_detailed(
        client=client,
to=to,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    to: str,

) -> Response[Error | FingerprintTransitionPreview]:
    """  Administrateurs uniquement. Aperçu d'impact du changement du modèle d'empreinte vers le modèle
    cible, sans rien modifier : références à recalculer, clips et Personnes concernés, autres empreintes
    à recalculer ensuite, attributions validées conservées, Workers qui ont déjà la cible prête (ADR
    0044).

    Args:
        to (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FingerprintTransitionPreview]
     """


    kwargs = _get_kwargs(
        to=to,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,
    to: str,

) -> Error | FingerprintTransitionPreview | None:
    """  Administrateurs uniquement. Aperçu d'impact du changement du modèle d'empreinte vers le modèle
    cible, sans rien modifier : références à recalculer, clips et Personnes concernés, autres empreintes
    à recalculer ensuite, attributions validées conservées, Workers qui ont déjà la cible prête (ADR
    0044).

    Args:
        to (str):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FingerprintTransitionPreview
     """


    return (await asyncio_detailed(
        client=client,
to=to,

    )).parsed
