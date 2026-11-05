from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.fingerprint_transition_input import FingerprintTransitionInput
from ...models.fingerprint_transition_state import FingerprintTransitionState
from typing import cast



def _get_kwargs(
    *,
    body: FingerprintTransitionInput,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/voice-models/fingerprint-transition",
    }

    _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | FingerprintTransitionState:
    if response.status_code == 201:
        response_201 = FingerprintTransitionState.from_dict(response.json())



        return response_201

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | FingerprintTransitionState]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,
    body: FingerprintTransitionInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | FingerprintTransitionState]:
    """  Administrateurs uniquement. Confirme le changement du modèle d'empreinte vers la cible.
    confirm_references doit être le nombre de références vu dans l'aperçu
    (voice_transition_preview_stale sinon). Une seule transition à la fois
    (voice_transition_in_progress).

    Args:
        origin (str):
        x_csrf_token (str):
        body (FingerprintTransitionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FingerprintTransitionState]
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
    client: AuthenticatedClient,
    body: FingerprintTransitionInput,
    origin: str,
    x_csrf_token: str,

) -> Error | FingerprintTransitionState | None:
    """  Administrateurs uniquement. Confirme le changement du modèle d'empreinte vers la cible.
    confirm_references doit être le nombre de références vu dans l'aperçu
    (voice_transition_preview_stale sinon). Une seule transition à la fois
    (voice_transition_in_progress).

    Args:
        origin (str):
        x_csrf_token (str):
        body (FingerprintTransitionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FingerprintTransitionState
     """


    return sync_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,
    body: FingerprintTransitionInput,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | FingerprintTransitionState]:
    """  Administrateurs uniquement. Confirme le changement du modèle d'empreinte vers la cible.
    confirm_references doit être le nombre de références vu dans l'aperçu
    (voice_transition_preview_stale sinon). Une seule transition à la fois
    (voice_transition_in_progress).

    Args:
        origin (str):
        x_csrf_token (str):
        body (FingerprintTransitionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FingerprintTransitionState]
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
    client: AuthenticatedClient,
    body: FingerprintTransitionInput,
    origin: str,
    x_csrf_token: str,

) -> Error | FingerprintTransitionState | None:
    """  Administrateurs uniquement. Confirme le changement du modèle d'empreinte vers la cible.
    confirm_references doit être le nombre de références vu dans l'aperçu
    (voice_transition_preview_stale sinon). Une seule transition à la fois
    (voice_transition_in_progress).

    Args:
        origin (str):
        x_csrf_token (str):
        body (FingerprintTransitionInput):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FingerprintTransitionState
     """


    return (await asyncio_detailed(
        client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
