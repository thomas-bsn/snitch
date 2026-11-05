from http import HTTPStatus
from typing import Any, cast
from urllib.parse import quote

import httpx

from ...client import AuthenticatedClient, Client
from ...types import Response, UNSET
from ... import errors

from ...models.error import Error
from ...models.worker_voice_models import WorkerVoiceModels
from typing import cast



def _get_kwargs(

) -> dict[str, Any]:






    _kwargs: dict[str, Any] = {
        "method": "get",
        "url": "/api/worker/voice-models",
    }


    return _kwargs



def _parse_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Error | WorkerVoiceModels:
    if response.status_code == 200:
        response_200 = WorkerVoiceModels.from_dict(response.json())



        return response_200

    response_default = Error.from_dict(response.json())



    return response_default



def _build_response(*, client: AuthenticatedClient | Client, response: httpx.Response) -> Response[Error | WorkerVoiceModels]:
    return Response(
        status_code=HTTPStatus(response.status_code),
        content=response.content,
        headers=response.headers,
        parsed=_parse_response(client=client, response=response),
    )


def sync_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | WorkerVoiceModels]:
    """  État souhaité des modèles vocaux tiré par le Worker : le candidat et le modèle actif à préparer, les
    modèles à supprimer et les ordres d'installation en cours. Le jeton Hugging Face central
    n'accompagne que la première livraison d'un ordre qui télécharge un modèle sous accès restreint (ADR
    0027) ; jamais de chemin.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerVoiceModels]
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

) -> Error | WorkerVoiceModels | None:
    """  État souhaité des modèles vocaux tiré par le Worker : le candidat et le modèle actif à préparer, les
    modèles à supprimer et les ordres d'installation en cours. Le jeton Hugging Face central
    n'accompagne que la première livraison d'un ordre qui télécharge un modèle sous accès restreint (ADR
    0027) ; jamais de chemin.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerVoiceModels
     """


    return sync_detailed(
        client=client,

    ).parsed

async def asyncio_detailed(
    *,
    client: AuthenticatedClient,

) -> Response[Error | WorkerVoiceModels]:
    """  État souhaité des modèles vocaux tiré par le Worker : le candidat et le modèle actif à préparer, les
    modèles à supprimer et les ordres d'installation en cours. Le jeton Hugging Face central
    n'accompagne que la première livraison d'un ordre qui télécharge un modèle sous accès restreint (ADR
    0027) ; jamais de chemin.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Response[Error | WorkerVoiceModels]
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

) -> Error | WorkerVoiceModels | None:
    """  État souhaité des modèles vocaux tiré par le Worker : le candidat et le modèle actif à préparer, les
    modèles à supprimer et les ordres d'installation en cours. Le jeton Hugging Face central
    n'accompagne que la première livraison d'un ordre qui télécharge un modèle sous accès restreint (ADR
    0027) ; jamais de chemin.

    Raises:
        errors.UnexpectedStatus: If the server returns an undocumented status code and Client.raise_on_unexpected_status is True.
        httpx.TimeoutException: If the request takes longer than Client.timeout.

    Returns:
        Error | WorkerVoiceModels
     """


    return (await asyncio_detailed(
        client=client,

    )).parsed
