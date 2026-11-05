from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.voice_model_order_input import VoiceModelOrderInput
from ...models.voice_model_retry import VoiceModelRetry
from ...models.worker_model_order import WorkerModelOrder
from ...types import UNSET, Unset
from typing import cast



def _get_kwargs(
    id: str,
    *,
    body: VoiceModelOrderInput | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> dict[str, Any]:
    headers: dict[str, Any] = {}
    headers["Origin"] = origin

    headers["X-CSRF-Token"] = x_csrf_token







    _kwargs: dict[str, Any] = {
        "method": "post",
        "url": "/api/admin/workers/{id}/voice-models/retry".format(id=quote(str(id), safe=""),),
    }


    if not isinstance(body, Unset):
        _kwargs["json"] = body.to_dict()

    headers["Content-Type"] = "application/json"

    _kwargs["headers"] = headers
    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | VoiceModelRetry | WorkerModelOrder:
    if response.status_code == 200:
        response_200 = VoiceModelRetry.from_dict(response.json())



        return response_200

    if response.status_code == 201:
        response_201 = WorkerModelOrder.from_dict(response.json())



        return response_201

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | VoiceModelRetry | WorkerModelOrder]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    id: str,
    *,
    client: AuthenticatedClient,
    body: VoiceModelOrderInput | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceModelRetry | WorkerModelOrder]:
    """  Administrateurs uniquement. Relance la préparation du candidat sur un Worker en échec (sans corps).
    Refusé (no_voice_candidate) sans candidat ou sans échec à relancer. Avec un corps {model_id,
    idempotency_key}, enregistre un nouvel ordre d'installation du modèle dont la dernière tentative a
    échoué sur ce Worker (201, voice_model_not_failed sinon) ; pour un remplacement dont le retrait de
    l'ancien a échoué, relance ce retrait (200).

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelOrderInput | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModelRetry | WorkerModelOrder]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = client.get_httpx_client().request(
        **kwargs,
    )

    return _build_response(client=client, response=response)

def sync(
    id: str,
    *,
    client: AuthenticatedClient,
    body: VoiceModelOrderInput | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceModelRetry | WorkerModelOrder | None:
    """  Administrateurs uniquement. Relance la préparation du candidat sur un Worker en échec (sans corps).
    Refusé (no_voice_candidate) sans candidat ou sans échec à relancer. Avec un corps {model_id,
    idempotency_key}, enregistre un nouvel ordre d'installation du modèle dont la dernière tentative a
    échoué sur ce Worker (201, voice_model_not_failed sinon) ; pour un remplacement dont le retrait de
    l'ancien a échoué, relance ce retrait (200).

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelOrderInput | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModelRetry | WorkerModelOrder
     """


    return sync_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    ).parsed

async def asyncio_detailed(
    id: str,
    *,
    client: AuthenticatedClient,
    body: VoiceModelOrderInput | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Response[Error | VoiceModelRetry | WorkerModelOrder]:
    """  Administrateurs uniquement. Relance la préparation du candidat sur un Worker en échec (sans corps).
    Refusé (no_voice_candidate) sans candidat ou sans échec à relancer. Avec un corps {model_id,
    idempotency_key}, enregistre un nouvel ordre d'installation du modèle dont la dernière tentative a
    échoué sur ce Worker (201, voice_model_not_failed sinon) ; pour un remplacement dont le retrait de
    l'ancien a échoué, relance ce retrait (200).

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelOrderInput | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | VoiceModelRetry | WorkerModelOrder]
     """


    kwargs = _get_kwargs(
        id=id,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )

    response = await client.get_async_httpx_client().request(
        **kwargs
    )

    return _build_response(client=client, response=response)

async def asyncio(
    id: str,
    *,
    client: AuthenticatedClient,
    body: VoiceModelOrderInput | Unset = UNSET,
    origin: str,
    x_csrf_token: str,

) -> Error | VoiceModelRetry | WorkerModelOrder | None:
    """  Administrateurs uniquement. Relance la préparation du candidat sur un Worker en échec (sans corps).
    Refusé (no_voice_candidate) sans candidat ou sans échec à relancer. Avec un corps {model_id,
    idempotency_key}, enregistre un nouvel ordre d'installation du modèle dont la dernière tentative a
    échoué sur ce Worker (201, voice_model_not_failed sinon) ; pour un remplacement dont le retrait de
    l'ancien a échoué, relance ce retrait (200).

    Args:
        id (str):
        origin (str):
        x_csrf_token (str):
        body (VoiceModelOrderInput | Unset):

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | VoiceModelRetry | WorkerModelOrder
     """


    return (await asyncio_detailed(
        id=id,
client=client,
body=body,
origin=origin,
x_csrf_token=x_csrf_token,

    )).parsed
