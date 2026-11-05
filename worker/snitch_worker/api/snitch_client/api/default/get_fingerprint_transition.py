from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.fingerprint_transition_state import FingerprintTransitionState
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/admin/voice-models/fingerprint-transition",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | FingerprintTransitionState:
    if response.status_code == 200:
        response_200 = FingerprintTransitionState.from_dict(response.json())



        return response_200

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

) -> Response[Error | FingerprintTransitionState]:
    """  Administrateurs uniquement. État de la transition d'empreinte ouverte ou, à défaut, de la dernière :
    phase, progression des références recalculées, tâches actives ou en échec, rattrapage restant. Lire
    l'état fait aussi avancer une transition interrompue.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FingerprintTransitionState]
     """


    kwargs = _get_kwargs(

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    *,
    client: AuthenticatedClient,

) -> Error | FingerprintTransitionState | None:
    """  Administrateurs uniquement. État de la transition d'empreinte ouverte ou, à défaut, de la dernière :
    phase, progression des références recalculées, tâches actives ou en échec, rattrapage restant. Lire
    l'état fait aussi avancer une transition interrompue.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FingerprintTransitionState
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | FingerprintTransitionState]:
    """  Administrateurs uniquement. État de la transition d'empreinte ouverte ou, à défaut, de la dernière :
    phase, progression des références recalculées, tâches actives ou en échec, rattrapage restant. Lire
    l'état fait aussi avancer une transition interrompue.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | FingerprintTransitionState]
     """


    kwargs = _get_kwargs(

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    *,
    client: AuthenticatedClient,

) -> Error | FingerprintTransitionState | None:
    """  Administrateurs uniquement. État de la transition d'empreinte ouverte ou, à défaut, de la dernière :
    phase, progression des références recalculées, tâches actives ou en échec, rattrapage restant. Lire
    l'état fait aussi avancer une transition interrompue.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | FingerprintTransitionState
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
